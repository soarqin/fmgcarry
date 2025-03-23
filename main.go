package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func main() {
	if len(os.Args) < 4 {
		println("Usage: fmgcarry <fmg1> <fmg2> <fmg3> [<txt>]")
		return
	}
	hasTxt := len(os.Args) > 4
	var err error
	if err = filepath.WalkDir(os.Args[1], func(path string, d os.DirEntry, err error) error {
		if d.IsDir() || filepath.Ext(path) != ".fmg" {
			return nil
		}
		print("Processing ", d.Name(), "...")
		var fmg1, fmg2, fmg3 *Fmg
		if fmg1, err = FmgLoad(path); err != nil {
			return err
		}
		if fmg2, err = FmgLoad(filepath.Join(os.Args[2], d.Name())); err != nil {
			return err
		}
		if fmg3, err = FmgLoad(filepath.Join(os.Args[3], d.Name())); err != nil {
			return err
		}
		dirty := false
		if hasTxt {
			txt, _ := loadTxt(filepath.Join(os.Args[4], d.Name()+".txt"))
			if len(fmg3.Text) < len(fmg2.Text) {
				fmg3.Text = append(fmg3.Text, make([]string, len(fmg2.Text)-len(fmg3.Text))...)
				dirty = true
			}
			for k, v := range fmg2.TextMap {
				if idx, ok := fmg1.TextMap[k]; ok {
					if isEmpty(fmg1.Text[idx]) && isEmpty(fmg2.Text[v]) {
						continue
					}
					if fmg1.Text[idx] != fmg2.Text[v] {
						fmg3.SetText(k, fmg2.Text[v])
						dirty = true
					}
				} else {
					fmg3.SetText(k, fmg2.Text[v])
					dirty = true
				}
			}
			for k, v := range txt {
				if v == "" {
					continue
				}
				fmg3.SetText(int32(k), v)
				dirty = true
			}
			if dirty {
				if err = fmg3.Save(); err != nil {
					return err
				}
			}
		} else {
			var of *os.File
			var err error
			if of, err = os.Create(d.Name() + ".txt"); err != nil {
				return err
			}
			defer of.Close()
			arr := make([]int, 0)
			for k, v := range fmg2.TextMap {
				if idx, ok := fmg1.TextMap[k]; ok {
					if isEmpty(fmg1.Text[idx]) && isEmpty(fmg2.Text[v]) {
						continue
					}
					if fmg1.Text[idx] != fmg2.Text[v] {
						arr = append(arr, int(k))
					}
				} else {
					arr = append(arr, int(k))
				}
			}
			sort.Ints(arr)
			for _, i := range arr {
				if idx, ok := fmg1.TextMap[int32(i)]; ok && !isEmpty(fmg1.Text[idx]) {
					if _, err = io.WriteString(of, "< "+strconv.Itoa(i)+":"+strconv.Quote(fmg1.Text[idx])+"\n"); err != nil {
						return err
					}
				}
				hasNew := false
				if idx, ok := fmg2.TextMap[int32(i)]; ok && !isEmpty(fmg2.Text[idx]) {
					if _, err = io.WriteString(of, "> "+strconv.Itoa(i)+":"+strconv.Quote(fmg2.Text[idx])+"\n"); err != nil {
						return err
					}
					hasNew = true
				}
				if idx, ok := fmg3.TextMap[int32(i)]; ok && !isEmpty(fmg3.Text[idx]) {
					if _, err = io.WriteString(of, "- "+strconv.Itoa(i)+":"+strconv.Quote(fmg3.Text[idx])+"\n"); err != nil {
						return err
					}
				}
				if hasNew {
					if _, err = io.WriteString(of, "= "+strconv.Itoa(i)+":\"\"\n"); err != nil {
						return err
					}
				}
				dirty = true
			}
			if !dirty {
				if err = of.Close(); err != nil {
					return err
				}
				if err = os.Remove(d.Name() + ".txt"); err != nil {
					return err
				}
			}
		}
		println("Done")
		return nil
	}); err != nil {
		panic(err)
	}
}

func isEmpty(s string) bool {
	return s == "" || s == "[ERROR]"
}

func loadTxt(filename string) (map[int]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	res := make(map[int]string)
	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0:2] != "= " {
			continue
		}
		line = line[2:]
		sep := strings.IndexRune(line, ':')
		if sep < 0 {
			continue
		}
		var text string
		if text, err = strconv.Unquote(strings.TrimSpace(line[sep+1:])); err != nil || text == "" {
			if err != nil {
				println("Error unquote: ", line)
			}
			continue
		}
		var idx int
		if idx, err = strconv.Atoi(line[:sep]); err != nil {
			continue
		}
		res[idx] = text
	}
	return res, nil
}
