//go:build ignore

package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// ★ コマンドが増えたらパス定数をここに追加する
const (
	versionFile = "./_cmd/prokishi-server/version" // マスタ
	configYml   = "./_cmd/prokishi-server/build/config.yml"
	packJsn     = "./_cmd/prokishi-server/frontend/package.json"

	// クライアント（prokishi）はサーバと同じバージョンで配る
	clientVersionFile = "./_cmd/prokishi/version"

	// 複数コマンドがある場合は追加する
	// subVersionFile = "./_cmd/sub/version"
	// subConfigYml   = "./_cmd/sub/build/config.yml"
	// subPackJsn     = "./_cmd/sub/frontend/package.json"

	// 行頭から空白だけで始まる行に限る（config.yml の ios 用のコメント行 `#   version: "..."` に一致させない）
	configRg  = `^\s*version:\s*"([0-9]+\.[0-9]+\.[0-9]+)"`
	configFmt = `  version: "%v"`
	packRg    = `"version":\s*"([0-9]+\.[0-9]+\.[0-9]+)"`
	packFmt   = `  "version": "%v",`
)

const inquiry = `
Now Version: %s

  Enter   -> Patch Version
  1:Patch -> %s
  2:Minor -> %s
  3:Major -> %s
  Other   -> Cancel

Please select the upgrade version(1-3)[1]:`

type ver struct{ major, minor, patch int }

func parseVer(v string) *ver {
	vals := strings.Split(v, ".")
	if len(vals) != 3 {
		return &ver{-1, -1, -1}
	}
	return &ver{parseInt(vals[0]), parseInt(vals[1]), parseInt(vals[2])}
}

func parseInt(v string) int {
	val, err := strconv.Atoi(v)
	if err != nil {
		return -1
	}
	return val
}

func (v ver) addPatch() *ver { return &ver{v.major, v.minor, v.patch + 1} }
func (v ver) addMinor() *ver { return &ver{v.major, v.minor + 1, 0} }
func (v ver) addMajor() *ver { return &ver{v.major + 1, 0, 0} }
func (v ver) String() string { return fmt.Sprintf("%d.%d.%d", v.major, v.minor, v.patch) }
func (v *ver) isError() bool { return v == nil || v.major == -1 }

var bump bool

func main() {
	flag.BoolVar(&bump, "bump", false, "対話的にバージョンを選択して更新")
	flag.Parse()
	if err := run(flag.Args()); err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)
		os.Exit(1)
	}
	fmt.Println("Success")
}

func run(args []string) error {
	now, err := parseVersion()
	if err != nil {
		return err
	}

	// 引数なし・フラグなし: 現在値で同期
	if len(args) == 0 && !bump {
		fmt.Println("Version:", now)
		return write(now)
	}

	var next *ver
	if bump {
		next = inquiryVersion(now)
	} else {
		next = parseVer(args[0])
	}
	if next.isError() {
		return fmt.Errorf("invalid version")
	}

	fmt.Println("Version:", next)

	// ★ 複数コマンドがある場合は versionFile をここに追加する
	for _, f := range []string{versionFile /*, subVersionFile */} {
		if err := os.WriteFile(f, []byte(next.String()), 0644); err != nil {
			return err
		}
		fmt.Println("Write:", f)
	}
	return write(next)
}

func inquiryVersion(now *ver) *ver {
	fmt.Fprintf(os.Stdout, inquiry,
		now.String(), now.addPatch(), now.addMinor(), now.addMajor())
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	switch scanner.Text() {
	case "", "1":
		return now.addPatch()
	case "2":
		return now.addMinor()
	case "3":
		return now.addMajor()
	}
	return &ver{-1, -1, -1}
}

func parseVersion() (*ver, error) {
	data, err := os.ReadFile(versionFile)
	if err != nil {
		return nil, err
	}
	return parseVer(strings.TrimSpace(string(data))), nil
}

type op struct {
	input, output string
	v             *ver
	rgs           []*rgSet
}

type rgSet struct {
	xp, format string
	rg         *regexp.Regexp
}

func write(v *ver) error {
	// マスタ以外の version ファイル（引数なしの同期でも書く）
	if err := os.WriteFile(clientVersionFile, []byte(v.String()), 0644); err != nil {
		return err
	}
	fmt.Println("Write:", clientVersionFile)

	// ★ 複数コマンドがある場合は ops にエントリを追加する
	ops := []*op{
		{configYml, "", v, []*rgSet{{configRg, configFmt, nil}}},
		{packJsn, "", v, []*rgSet{{packRg, packFmt, nil}}},
		// {subConfigYml, "", v, []*rgSet{{configRg, configFmt, nil}}},
		// {subPackJsn,   "", v, []*rgSet{{packRg,   packFmt,   nil}}},
	}
	for _, o := range ops {
		if err := writeFile(o); err != nil {
			return err
		}
	}
	for _, o := range ops {
		os.Rename(o.output, o.input)
		fmt.Println("Rename:", o.input)
	}
	return nil
}

func writeFile(o *op) error {
	for _, s := range o.rgs {
		s.rg = regexp.MustCompile(s.xp)
	}
	in, err := os.Open(o.input)
	if err != nil {
		return err
	}
	defer in.Close()

	o.output = o.input + "_tmp"
	out, err := os.Create(o.output)
	if err != nil {
		return err
	}
	defer out.Close()

	fmt.Println("Write temp:", o.output)
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		line := scanner.Text()
		for _, s := range o.rgs {
			if m := s.rg.FindStringSubmatch(line); len(m) > 1 {
				line = fmt.Sprintf(s.format, o.v)
				break
			}
		}
		fmt.Fprintln(out, line)
	}
	return scanner.Err()
}
