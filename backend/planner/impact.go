package planner

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// grepCallersTimeout bounds the caller-search grep so a huge repo or slow
// filesystem can't hang plan generation indefinitely -- this runs inside
// the async worker (FulfillPlan), which has no other way to cancel it.
const grepCallersTimeout = 15 * time.Second

// analyzeImpact finds the function declaration enclosing line in
// repoRef/filePath (go/ast, per cmd/lint-fixed-plan.md Rule 7's own
// "AST ဖြင့် analyze" spec) and greps repoRef for other call sites of that
// function's name. Best-effort throughout, same spirit as
// before-fixed/*.md §7's own honest "No external callers found" wording --
// a parse failure, no enclosing function, or zero grep matches are all
// reported as plain findings, never errors.
func analyzeImpact(ctx context.Context, repoRef, filePath string, line int) ImpactInfo {
	info := ImpactInfo{AffectedFile: filePath}

	full, err := resolveInRepo(repoRef, filePath)
	if err != nil {
		info.Callers = []string{"could not resolve file for impact analysis: " + err.Error()}
		return info
	}
	fset := token.NewFileSet()
	astFile, err := parser.ParseFile(fset, full, nil, parser.ParseComments)
	if err != nil {
		info.Callers = []string{"could not parse file for impact analysis: " + err.Error()}
		return info
	}
	info.AffectedPackage = astFile.Name.Name

	funcName, declLine := enclosingFuncName(fset, astFile, line)
	if funcName == "" {
		info.Callers = []string{"no callers found — declaration is not inside a function"}
		return info
	}
	info.AffectedSymbol = funcName

	callers, err := grepCallers(ctx, repoRef, full, declLine, funcName)
	if err != nil {
		info.Callers = []string{"caller search failed: " + err.Error()}
		return info
	}
	if len(callers) == 0 {
		info.Callers = []string{"no callers found — change is local to " + astFile.Name.Name}
		return info
	}
	info.Callers = callers
	return info
}

// enclosingFuncName returns the name of the *ast.FuncDecl (method or
// plain function) whose body contains line, and the line its own "func
// Name(...)" header sits on (so grepCallers can exclude just that one
// declaration line, not the whole file -- callers in the same file, e.g.
// a small main.go, are common and must still be found). Returns ("", 0)
// if line falls outside every function (e.g. a package-level var/const).
func enclosingFuncName(fset *token.FileSet, f *ast.File, line int) (name string, declLine int) {
	ast.Inspect(f, func(n ast.Node) bool {
		fd, ok := n.(*ast.FuncDecl)
		if !ok {
			return true
		}
		startLine := fset.Position(fd.Pos()).Line
		endLine := fset.Position(fd.End()).Line
		if line >= startLine && line <= endLine {
			name = fd.Name.Name
			declLine = startLine
			return false
		}
		return true
	})
	return name, declLine
}

// grepCallers shells out to grep (already a hard dependency of this
// binary's runtime image, same as git/golangci-lint in scanner.go/
// fixer.go) for call sites of funcName as a whole word under repoRef,
// excluding only the exact definingFile:definingLine match (the "func
// Name(...)" header's own line) -- not the whole file, since callers in
// the same file (a small main.go calling a helper it declares) are the
// common case and must still be found. A real reference vs. an
// unrelated same-named identifier can't be told apart by grep alone --
// this is the same best-effort tradeoff before-fixed/*.md's own Rule 7
// accepts ("Callers: grep -rn <function_name> ဖြင့် reference ရှာ").
// Bounded by grepCallersTimeout -- see its own doc comment for why
// (matches scanner.Run/fixer.Apply's existing exec.CommandContext usage
// against the same scanned repos).
func grepCallers(
	ctx context.Context,
	repoRef, definingFile string,
	definingLine int,
	funcName string,
) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, grepCallersTimeout)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"grep",
		"-rn",
		"--include=*.go",
		`\b`+funcName+`\b`,
		repoRef,
	)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil // grep exit 1 = no matches, not a failure
		}
		return nil, err
	}

	var callers []string
	for line := range strings.SplitSeq(strings.TrimRight(string(out), "\n"), "\n") {
		if line == "" {
			continue
		}
		path, rest, _ := strings.Cut(line, ":")
		lineNoStr, _, _ := strings.Cut(rest, ":")
		abs, err := filepath.Abs(path)
		if err == nil && abs == definingFile && lineNoStr == strconv.Itoa(definingLine) {
			continue // the declaration's own header line, not a caller
		}
		callers = append(callers, strings.TrimPrefix(line, repoRef+"/"))
	}
	return callers, nil
}
