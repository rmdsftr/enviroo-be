package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

const (
	cReset     = "\033[0m"
	cBold      = "\033[1m"
	cRed       = "\033[31m"
	cGreen     = "\033[32m"
	cYellow    = "\033[33m"
	cCyan      = "\033[36m"
	cGray      = "\033[90m"
	cBoldRed   = "\033[1;31m"
	cBoldGreen = "\033[1;32m"
	cBoldYel   = "\033[1;33m"
)

var useColor = os.Getenv("NO_COLOR") == ""

// File test yang dilewati: tidak dijalankan, dan file sumber pasangannya tidak dihitung di coverage.
var excludedTests = map[string]bool{
	"internal/services/distribusi_sisa_service_test.go": true,
	"internal/services/pengangkutan_service_test.go":    true,
}

func excludedSource(name string) bool {
	return excludedTests[strings.TrimSuffix(name, ".go")+"_test.go"]
}

func paint(code, s string) string {
	if !useColor {
		return s
	}
	return code + s + cReset
}

type testFile struct {
	rel      string
	dir      string
	tests    []string
	parseErr string
}

type testEvent struct {
	Action      string
	Package     string
	Test        string
	Elapsed     float64
	Output      string
	FailedBuild string
}

type caseResult struct {
	name    string
	status  string
	elapsed float64
	output  []string
}

type block struct {
	stmts   int
	covered bool
}

type coverProfile map[string]map[string]*block

type runResult struct {
	file      testFile
	cases     []*caseResult
	buildOut  []string
	pkgOut    []string
	stderr    []string
	buildFail bool
	pkgFail   bool
	execErr   string
	profile   coverProfile
}

type testBinary struct {
	path     string
	buildErr []string
	execErr  string
}

type covEntry struct {
	file           string
	covered, total int
}

func main() {
	os.Exit(run())
}

func run() int {
	start := time.Now()

	root, module, err := findRoot()
	if err != nil {
		fmt.Fprintln(os.Stderr, paint(cBoldRed, err.Error()))
		return 1
	}

	files, err := collectTestFiles(root)
	if err != nil {
		fmt.Fprintln(os.Stderr, paint(cBoldRed, "Gagal membaca file test: "+err.Error()))
		return 1
	}
	if len(files) == 0 {
		fmt.Println("Tidak ada file test di internal/")
		return 0
	}

	tmpDir, err := os.MkdirTemp("", "unittest-cover-")
	if err != nil {
		fmt.Fprintln(os.Stderr, paint(cBoldRed, "Gagal membuat folder sementara: "+err.Error()))
		return 1
	}
	defer os.RemoveAll(tmpDir)

	line := strings.Repeat("═", 64)
	fmt.Println(paint(cBold, line))
	fmt.Println(paint(cBold, fmt.Sprintf("  UNIT TEST · %s · %d file test", module, len(files))))
	fmt.Println(paint(cBold, line))

	merged := coverProfile{}
	results := make([]*runResult, 0, len(files))
	bins := map[string]testBinary{}
	for i, tf := range files {
		bin, built := bins[tf.dir]
		if !built && tf.parseErr == "" && len(tf.tests) > 0 {
			fmt.Printf("\n%s\n", paint(cGray, "  compile test binary "+tf.dir+" ..."))
			bin = buildBinary(root, tf.dir, tmpDir, len(bins))
			bins[tf.dir] = bin
		}
		fmt.Printf("\n%s %s\n", paint(cCyan, fmt.Sprintf("[%d/%d]", i+1, len(files))), paint(cBold, tf.rel))
		r := runFile(root, module, tf, bin, tmpDir, i)
		printRun(r)
		merged.merge(r.profile)
		results = append(results, r)
	}

	if !printSummary(root, results, merged, time.Since(start)) {
		return 1
	}
	return 0
}

func findRoot() (string, string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", "", err
	}
	for {
		data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
		if err == nil {
			for _, l := range strings.Split(string(data), "\n") {
				l = strings.TrimSpace(l)
				if strings.HasPrefix(l, "module ") {
					return dir, strings.Trim(strings.TrimSpace(strings.TrimPrefix(l, "module ")), `"`), nil
				}
			}
			return "", "", fmt.Errorf("baris module tidak ditemukan di %s", filepath.Join(dir, "go.mod"))
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", "", errors.New("go.mod tidak ditemukan, jalankan dari dalam repo")
		}
		dir = parent
	}
}

func collectTestFiles(root string) ([]testFile, error) {
	var files []testFile
	fset := token.NewFileSet()
	err := filepath.WalkDir(filepath.Join(root, "internal"), func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if excludedTests[rel] {
			return nil
		}
		tf := testFile{rel: rel, dir: path.Dir(rel)}

		f, err := parser.ParseFile(fset, p, nil, parser.SkipObjectResolution)
		if err != nil {
			tf.parseErr = err.Error()
			files = append(files, tf)
			return nil
		}
		for _, decl := range f.Decls {
			if fd, ok := decl.(*ast.FuncDecl); ok && isTestFunc(fd) {
				tf.tests = append(tf.tests, fd.Name.Name)
			}
		}
		files = append(files, tf)
		return nil
	})
	sort.Slice(files, func(i, j int) bool { return files[i].rel < files[j].rel })
	return files, err
}

func isTestFunc(fd *ast.FuncDecl) bool {
	if fd.Recv != nil || fd.Type.TypeParams != nil || !strings.HasPrefix(fd.Name.Name, "Test") {
		return false
	}
	if rest := fd.Name.Name[len("Test"):]; rest != "" {
		if r, _ := utf8.DecodeRuneInString(rest); unicode.IsLower(r) {
			return false
		}
	}
	params := fd.Type.Params
	if params == nil || len(params.List) != 1 || len(params.List[0].Names) > 1 {
		return false
	}
	star, ok := params.List[0].Type.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := star.X.(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "T"
}

func splitLines(s string) []string {
	var out []string
	for _, l := range strings.Split(strings.TrimRight(s, "\r\n"), "\n") {
		if l = strings.TrimRight(l, "\r"); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func buildBinary(root, dir, tmpDir string, idx int) testBinary {
	name := filepath.Join(tmpDir, fmt.Sprintf("pkg-%d.test", idx))
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	cmd := exec.Command("go", "test", "-c", "-cover", "-o", name, "./"+dir)
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			return testBinary{execErr: err.Error()}
		}
		return testBinary{buildErr: splitLines(string(out))}
	}
	return testBinary{path: name}
}

func runFile(root, module string, tf testFile, bin testBinary, tmpDir string, idx int) *runResult {
	r := &runResult{file: tf}
	switch {
	case tf.parseErr != "" || len(tf.tests) == 0:
		return r
	case bin.execErr != "":
		r.execErr = bin.execErr
		return r
	case bin.path == "":
		r.buildFail = true
		r.buildOut = bin.buildErr
		return r
	}

	prof := filepath.Join(tmpDir, fmt.Sprintf("cover-%d.out", idx))
	pattern := "^(" + strings.Join(tf.tests, "|") + ")$"
	cmd := exec.Command("go", "tool", "test2json", "-p", module+"/"+tf.dir, bin.path,
		"-test.v=test2json", "-test.run", pattern, "-test.coverprofile="+prof,
		"-test.timeout=10m", "-test.paniconexit0")
	cmd.Dir = filepath.Join(root, filepath.FromSlash(tf.dir))
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, err := cmd.Output()

	byName := map[string]*caseResult{}
	sc := bufio.NewScanner(bytes.NewReader(stdout))
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		var ev testEvent
		if json.Unmarshal(sc.Bytes(), &ev) != nil {
			r.pkgOut = append(r.pkgOut, sc.Text())
			continue
		}
		out := strings.TrimRight(ev.Output, "\r\n")
		switch {
		case ev.Action == "build-output":
			r.buildOut = append(r.buildOut, out)
		case ev.Action == "build-fail":
			r.buildFail = true
		case ev.Test == "":
			switch ev.Action {
			case "output":
				r.pkgOut = append(r.pkgOut, out)
			case "fail":
				r.pkgFail = true
				if ev.FailedBuild != "" {
					r.buildFail = true
				}
			}
		default:
			c := byName[ev.Test]
			if c == nil {
				c = &caseResult{name: ev.Test}
				byName[ev.Test] = c
				r.cases = append(r.cases, c)
			}
			switch ev.Action {
			case "output":
				c.output = append(c.output, out)
			case "pass", "fail", "skip":
				c.status = ev.Action
				c.elapsed = ev.Elapsed
			}
		}
	}

	r.stderr = splitLines(stderr.String())

	var exitErr *exec.ExitError
	if err != nil && !errors.As(err, &exitErr) {
		r.execErr = err.Error()
	}
	if err != nil && r.execErr == "" && !r.pkgFail && !r.buildFail {
		r.pkgFail = true
	}

	r.profile = parseProfile(prof, module)
	os.Remove(prof)
	return r
}

func parseProfile(file, module string) coverProfile {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil
	}
	p := coverProfile{}
	for _, l := range strings.Split(string(data), "\n") {
		l = strings.TrimSpace(l)
		if l == "" || strings.HasPrefix(l, "mode:") {
			continue
		}
		colon := strings.LastIndexByte(l, ':')
		if colon < 0 {
			continue
		}
		fields := strings.Fields(l[colon+1:])
		if len(fields) != 3 {
			continue
		}
		stmts, err1 := strconv.Atoi(fields[1])
		count, err2 := strconv.Atoi(fields[2])
		if err1 != nil || err2 != nil {
			continue
		}
		name := strings.TrimPrefix(l[:colon], module+"/")
		if excludedSource(name) {
			continue
		}
		if p[name] == nil {
			p[name] = map[string]*block{}
		}
		b := p[name][fields[0]]
		if b == nil {
			b = &block{stmts: stmts}
			p[name][fields[0]] = b
		}
		if count > 0 {
			b.covered = true
		}
	}
	return p
}

func (p coverProfile) merge(src coverProfile) {
	for name, blocks := range src {
		if p[name] == nil {
			p[name] = map[string]*block{}
		}
		for pos, b := range blocks {
			dst := p[name][pos]
			if dst == nil {
				p[name][pos] = &block{stmts: b.stmts, covered: b.covered}
				continue
			}
			dst.covered = dst.covered || b.covered
		}
	}
}

func sumBlocks(blocks map[string]*block) (covered, total int) {
	for _, b := range blocks {
		total += b.stmts
		if b.covered {
			covered += b.stmts
		}
	}
	return covered, total
}

func (r *runResult) counts() (pass, fail, skip int) {
	for _, c := range r.cases {
		switch c.status {
		case "pass":
			pass++
		case "skip":
			skip++
		default:
			fail++
		}
	}
	return pass, fail, skip
}

func (r *runResult) failed() bool {
	_, fail, _ := r.counts()
	return fail > 0 || r.buildFail || r.pkgFail || r.execErr != "" || r.parseErr()
}

func (r *runResult) parseErr() bool { return r.file.parseErr != "" }

func (r *runResult) skipped() bool { return !r.parseErr() && len(r.file.tests) == 0 }

func (r *runResult) unitCoverage() ([]covEntry, bool) {
	if r.profile == nil {
		return nil, false
	}
	src := path.Join(r.file.dir, strings.TrimSuffix(path.Base(r.file.rel), "_test.go")+".go")
	if blocks, ok := r.profile[src]; ok {
		c, t := sumBlocks(blocks)
		return []covEntry{{src, c, t}}, true
	}
	var entries []covEntry
	for name, blocks := range r.profile {
		if c, t := sumBlocks(blocks); c > 0 {
			entries = append(entries, covEntry{name, c, t})
		}
	}
	sort.Slice(entries, func(i, j int) bool {
		pi, pj := pct(entries[i].covered, entries[i].total), pct(entries[j].covered, entries[j].total)
		if pi != pj {
			return pi > pj
		}
		return entries[i].file < entries[j].file
	})
	return entries, false
}

func depth(name string) int { return strings.Count(name, "/") }

func statusLabel(s string) string {
	switch s {
	case "pass":
		return paint(cBoldGreen, "PASS")
	case "fail":
		return paint(cBoldRed, "FAIL")
	case "skip":
		return paint(cBoldYel, "SKIP")
	default:
		return paint(cBoldRed, "????")
	}
}

func detailLines(lines []string) []string {
	var out []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "=== RUN") || strings.HasPrefix(t, "=== PAUSE") ||
			strings.HasPrefix(t, "=== CONT") || strings.HasPrefix(t, "=== NAME") ||
			strings.HasPrefix(t, "--- PASS") || strings.HasPrefix(t, "--- FAIL") ||
			strings.HasPrefix(t, "--- SKIP") {
			continue
		}
		out = append(out, strings.TrimPrefix(l, "    "))
	}
	return out
}

func pkgDetailLines(lines []string) []string {
	var out []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t == "" || t == "PASS" || t == "FAIL" || strings.HasPrefix(t, "ok ") ||
			strings.HasPrefix(t, "FAIL\t") || strings.HasPrefix(t, "coverage:") {
			continue
		}
		out = append(out, l)
	}
	return out
}

func pct(covered, total int) float64 {
	if total == 0 {
		return 0
	}
	return float64(covered) / float64(total) * 100
}

func pctColor(p float64) string {
	switch {
	case p >= 80:
		return cGreen
	case p >= 50:
		return cYellow
	default:
		return cRed
	}
}

func pctText(covered, total int) string {
	if total == 0 {
		return paint(cGray, "    -")
	}
	p := pct(covered, total)
	return paint(pctColor(p), fmt.Sprintf("%5.1f%%", p))
}

func bar(covered, total int) string {
	const width = 20
	if total == 0 {
		return paint(cGray, strings.Repeat("░", width))
	}
	p := pct(covered, total)
	n := int(p/100*width + 0.5)
	if n > width {
		n = width
	}
	return paint(pctColor(p), strings.Repeat("█", n)) + paint(cGray, strings.Repeat("░", width-n))
}

func printRun(r *runResult) {
	switch {
	case r.parseErr():
		fmt.Println("  " + paint(cBoldRed, "GAGAL PARSE") + "  " + r.file.parseErr)
		return
	case r.skipped():
		fmt.Println("  " + paint(cGray, "tidak ada fungsi Test, dilewati"))
		return
	case r.execErr != "":
		fmt.Println("  " + paint(cBoldRed, "GAGAL MENJALANKAN go test") + "  " + r.execErr)
		return
	case r.buildFail:
		fmt.Println("  " + paint(cBoldRed, "BUILD GAGAL"))
		for _, l := range append(r.buildOut, r.stderr...) {
			fmt.Println("      " + l)
		}
		return
	}

	width := 0
	for _, c := range r.cases {
		if w := 2*depth(c.name) + len(c.name); w > width {
			width = w
		}
	}
	for _, c := range r.cases {
		indent := strings.Repeat("  ", depth(c.name))
		pad := strings.Repeat(" ", width-len(indent)-len(c.name))
		fmt.Printf("  %s  %s%s%s  %s\n", statusLabel(c.status), indent, c.name, pad, paint(cGray, fmt.Sprintf("%.2fs", c.elapsed)))
		if c.status == "pass" {
			continue
		}
		for _, l := range detailLines(c.output) {
			if c.status == "skip" {
				l = paint(cGray, l)
			}
			fmt.Printf("        %s%s\n", indent, l)
		}
	}

	pass, fail, skip := r.counts()
	if r.pkgFail && fail == 0 {
		fmt.Println("  " + paint(cBoldRed, "PACKAGE GAGAL") + paint(cGray, " (gagal di luar test case)"))
		for _, l := range append(pkgDetailLines(r.pkgOut), r.stderr...) {
			fmt.Println("      " + l)
		}
	}

	failText := paint(cGray, "0 fail")
	if fail > 0 {
		failText = paint(cBoldRed, fmt.Sprintf("%d fail", fail))
	}
	counts := fmt.Sprintf("%s · %s · %s", paint(cGreen, fmt.Sprintf("%d pass", pass)), failText, paint(cYellow, fmt.Sprintf("%d skip", skip)))
	fmt.Printf("  %s %s %s %s\n", paint(cCyan, "→"), counts, paint(cGray, "|"), unitCoverageText(r))
}

func unitCoverageText(r *runResult) string {
	entries, paired := r.unitCoverage()
	switch {
	case r.profile == nil:
		return paint(cGray, "coverage: tidak ada data")
	case paired:
		e := entries[0]
		return fmt.Sprintf("coverage %s: %s %s", path.Base(e.file), pctText(e.covered, e.total), paint(cGray, fmt.Sprintf("(%d/%d)", e.covered, e.total)))
	case len(entries) == 0:
		return paint(cGray, "coverage: tidak ada statement yang tersentuh")
	}
	parts := make([]string, 0, len(entries))
	for _, e := range entries {
		parts = append(parts, fmt.Sprintf("%s %s %s", path.Base(e.file), pctText(e.covered, e.total), paint(cGray, fmt.Sprintf("(%d/%d)", e.covered, e.total))))
	}
	return "coverage (tanpa file pasangan): " + strings.Join(parts, ", ")
}

func section(title string) {
	fmt.Println()
	fmt.Println(paint(cBold, "  "+title))
	fmt.Println(paint(cGray, "  "+strings.Repeat("─", 62)))
}

func short(rel string) string { return strings.TrimPrefix(rel, "internal/") }

func printSummary(root string, results []*runResult, merged coverProfile, elapsed time.Duration) bool {
	line := strings.Repeat("═", 64)
	fmt.Println()
	fmt.Println(paint(cBold, line))
	fmt.Println(paint(cBold, "  RINGKASAN"))
	fmt.Println(paint(cBold, line))

	var pass, fail, skip, filesOK, filesFail int
	var failedList []string
	tested := map[string]bool{}
	for _, r := range results {
		if r.skipped() {
			continue
		}
		tested[r.file.dir] = true
		p, f, s := r.counts()
		pass, fail, skip = pass+p, fail+f, skip+s
		if r.failed() {
			filesFail++
		} else {
			filesOK++
		}
		switch {
		case r.parseErr():
			failedList = append(failedList, r.file.rel+" › (gagal parse)")
		case r.execErr != "":
			failedList = append(failedList, r.file.rel+" › (go test gagal dijalankan)")
		case r.buildFail:
			failedList = append(failedList, r.file.rel+" › (build gagal)")
		default:
			for _, c := range r.cases {
				if c.status != "pass" && c.status != "skip" {
					failedList = append(failedList, r.file.rel+" › "+c.name)
				}
			}
			if r.pkgFail && f == 0 {
				failedList = append(failedList, r.file.rel+" › (package gagal di luar test case)")
			}
		}
	}

	failText := paint(cGray, "0 fail")
	if fail > 0 {
		failText = paint(cBoldRed, fmt.Sprintf("%d fail", fail))
	}
	fileFailText := paint(cGray, "0 ada yang gagal")
	if filesFail > 0 {
		fileFailText = paint(cBoldRed, fmt.Sprintf("%d ada yang gagal", filesFail))
	}
	fmt.Printf("  Test case : %d total · %s · %s · %s\n", pass+fail+skip, paint(cGreen, fmt.Sprintf("%d pass", pass)), failText, paint(cYellow, fmt.Sprintf("%d skip", skip)))
	fmt.Printf("  File test : %d total · %s · %s\n", filesOK+filesFail, paint(cGreen, fmt.Sprintf("%d lulus semua", filesOK)), fileFailText)
	fmt.Printf("  Durasi    : %s\n", elapsed.Round(100*time.Millisecond))

	type row struct {
		label          string
		covered, total int
		note           string
	}
	var unitRows []row
	for _, r := range results {
		if r.skipped() {
			continue
		}
		entries, paired := r.unitCoverage()
		switch {
		case r.profile == nil:
			unitRows = append(unitRows, row{label: short(r.file.rel), note: "tidak ada data coverage"})
		case paired:
			unitRows = append(unitRows, row{label: short(r.file.rel) + " → " + path.Base(entries[0].file), covered: entries[0].covered, total: entries[0].total})
		case len(entries) == 0:
			unitRows = append(unitRows, row{label: short(r.file.rel), note: "tidak ada statement yang tersentuh"})
		default:
			for _, e := range entries {
				unitRows = append(unitRows, row{label: short(r.file.rel) + " → " + path.Base(e.file) + " (tersentuh)", covered: e.covered, total: e.total})
			}
		}
	}
	printRows := func(rows []row) {
		width := 0
		for _, rw := range rows {
			if n := utf8.RuneCountInString(rw.label); n > width {
				width = n
			}
		}
		for _, rw := range rows {
			pad := strings.Repeat(" ", width-utf8.RuneCountInString(rw.label))
			if rw.note != "" {
				fmt.Printf("  %s%s  %s\n", rw.label, pad, paint(cGray, rw.note))
				continue
			}
			fmt.Printf("  %s%s  %s  %s  %s\n", rw.label, pad, pctText(rw.covered, rw.total), bar(rw.covered, rw.total), paint(cGray, fmt.Sprintf("(%d/%d)", rw.covered, rw.total)))
		}
	}

	section("Coverage per unit test (file test → file sumber)")
	printRows(unitRows)

	pkgCov := map[string][2]int{}
	for name, blocks := range merged {
		c, t := sumBlocks(blocks)
		v := pkgCov[path.Dir(name)]
		pkgCov[path.Dir(name)] = [2]int{v[0] + c, v[1] + t}
	}
	pkgs := make([]string, 0, len(pkgCov))
	for p := range pkgCov {
		pkgs = append(pkgs, p)
	}
	sort.Strings(pkgs)
	var pkgRows []row
	var totalCovered, totalStmts int
	for _, p := range pkgs {
		v := pkgCov[p]
		totalCovered, totalStmts = totalCovered+v[0], totalStmts+v[1]
		pkgRows = append(pkgRows, row{label: p, covered: v[0], total: v[1]})
	}
	pkgRows = append(pkgRows, row{label: "TOTAL", covered: totalCovered, total: totalStmts})

	section("Coverage per package")
	printRows(pkgRows)

	if untested := untestedPackages(root, tested); len(untested) > 0 {
		section("Package tanpa test (tidak dihitung di TOTAL)")
		for _, p := range untested {
			fmt.Println("  " + paint(cGray, p))
		}
	}

	if len(failedList) > 0 {
		section("Test yang gagal")
		for _, f := range failedList {
			fmt.Println("  " + paint(cRed, "✘ "+f))
		}
	}

	fmt.Println()
	if filesFail > 0 {
		fmt.Println(paint(cBoldRed, "  ADA TEST YANG GAGAL"))
		return false
	}
	fmt.Println(paint(cBoldGreen, "  SEMUA TEST LULUS"))
	return true
}

func untestedPackages(root string, tested map[string]bool) []string {
	var out []string
	filepath.WalkDir(filepath.Join(root, "internal"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if d.Name() == "mocks" {
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if tested[rel] {
			return nil
		}
		entries, err := os.ReadDir(p)
		if err != nil {
			return nil
		}
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".go") && !strings.HasSuffix(e.Name(), "_test.go") {
				out = append(out, rel)
				break
			}
		}
		return nil
	})
	return out
}
