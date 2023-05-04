package main

import (
	"flag"
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"runtime/pprof"
	"strings"
	"sync"

	"golang.org/x/tools/go/callgraph"
	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/pointer"
	"golang.org/x/tools/go/ssa"
	"golang.org/x/tools/go/ssa/ssautil"
)

func main() {
	flag.Parse()
	// 创建cpu分析文件
	if true {
		f, err := os.Create("./main-cpu.prof")
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	//生成Go Packages
	cfg := &packages.Config{Mode: packages.LoadAllSyntax}
	pkgs, err := packages.Load(cfg, flag.Args()...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load: %v\n", err)
		os.Exit(1)
	}
	if packages.PrintErrors(pkgs) > 0 {
		os.Exit(1)
	}

	//生成ssa
	prog, pkgs1 := ssautil.AllPackages(pkgs, 0)
	prog.Build()
	//找出main package
	mains, err := mainPackages(pkgs1)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//使用pointer生成调用链路
	config := &pointer.Config{
		Mains:          mains,
		BuildCallGraph: true,
	}
	result, err := pointer.Analyze(config)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	//遍历调用链路
	// callgraph.GraphVisitEdges(result.CallGraph, func(edge *callgraph.Edge) error {
	GraphVisitEdges(result.CallGraph, func(edge *callgraph.Edge) error {

		//过滤非源代码
		// if isSynthetic(edge) {
		// 	return nil
		// }

		caller := edge.Caller
		callee := edge.Callee

		//过滤标准库代码
		// if inStd(caller) || inStd(callee) {
		// 	return nil
		// }
		//过滤其他package TODO
		// limits := []string{"go-tools/go-callvis/test"}
		// if !inLimits(caller, limits) || !inLimits(callee, limits) {
		// 	return nil
		// }

		posCaller := prog.Fset.Position(caller.Func.Pos())
		filenameCaller := filepath.Base(posCaller.Filename)

		//输出调用信息
		fmt.Fprintf(os.Stdout, "call node: %s -> %s (%s -> %s) %v\n", caller.Func.Pkg, callee.Func.Pkg, caller, callee, filenameCaller)
		return nil
	})
}
func GraphVisitEdges1(g *callgraph.Graph, edge func(*callgraph.Edge) error) error {
	// seen := make(map[*callgraph.Node]struct{})
	var seen sync.Map
	var visit func(n *callgraph.Node) error
	visit = func(n *callgraph.Node) error {
		if _, ok := seen.Load(n); ok {
			return nil

		}
		seen.Store(n, struct{}{})
		for _, e := range n.Out {
			if err := visit(e.Callee); err != nil {
				return err
			}
			if err := edge(e); err != nil {
				return err
			}
		}

		return nil
	}
	var wg sync.WaitGroup
	for _, n := range g.Nodes {
		wg.Add(1)
		v := n
		go func() {
			defer wg.Done()
			if err := visit(v); err != nil {
				return
			}
		}()

	}
	wg.Wait()
	return nil
}
func GraphVisitEdges(g *callgraph.Graph, edge func(*callgraph.Edge) error) error {
	var seen sync.Map
	var visit func(n *callgraph.Node) error
	visit = func(n *callgraph.Node) error {
		if _, ok := seen.Load(n); ok {
			return nil

		}
		seen.Store(n, struct{}{})
		var wg sync.WaitGroup
		for _, e := range n.Out {
			wg.Add(1)
			go func(e *callgraph.Edge) {
				defer wg.Done()
				if err := visit(e.Callee); err != nil {
					return
				}
				if err := edge(e); err != nil {
					return
				}
			}(e)
		}
		wg.Wait()
		return nil
	}
	for _, n := range g.Nodes {
		if err := visit(n); err != nil {
			return err
		}
	}
	return nil
}

func mainPackages(pkgs []*ssa.Package) ([]*ssa.Package, error) {
	var mains []*ssa.Package
	for _, p := range pkgs {
		if p != nil && p.Pkg.Name() == "main" && p.Func("main") != nil {
			mains = append(mains, p)
		}
	}
	if len(mains) == 0 {
		return nil, fmt.Errorf("no main packages")
	}
	return mains, nil
}

func isSynthetic(edge *callgraph.Edge) bool {
	return edge.Caller.Func.Pkg == nil || edge.Callee.Func.Synthetic != ""
}

func inStd(node *callgraph.Node) bool {
	pkg, _ := build.Import(node.Func.Pkg.Pkg.Path(), "", 0)
	return pkg.Goroot
}

func inLimits(node *callgraph.Node, limitPaths []string) bool {
	pkgPath := node.Func.Pkg.Pkg.Path()
	for _, p := range limitPaths {
		if strings.HasPrefix(pkgPath, p) {
			return true
		}
	}
	return false
}
