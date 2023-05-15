package main

import (
	"bufio"
	"io"
	"os"
	"path/filepath"
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
		m := min(len(fmg1.Text), len(fmg2.Text))
		if hasTxt {
			txt, _ := loadTxt(filepath.Join(os.Args[4], d.Name()+".txt"))
			if len(fmg3.Text) < len(fmg2.Text) {
				fmg3.Text = append(fmg3.Text, make([]string, len(fmg2.Text)-len(fmg3.Text))...)
				dirty = true
			}
			for i := 0; i < m; i++ {
				if isEmpty(fmg1.Text[i]) && isEmpty(fmg2.Text[i]) {
					continue
				}
				if fmg1.Text[i] != fmg2.Text[i] {
					if text, found := txt[i]; found {
						fmg3.Text[i] = text
					} else {
						fmg3.Text[i] = fmg2.Text[i]
					}
					dirty = true
				}
			}
			if len(fmg2.Text) > m {
				copy(fmg3.Text[m:], fmg2.Text[m:])
				for i := m; i < len(fmg2.Text); i++ {
					if text, found := txt[i]; found {
						fmg3.Text[i] = text
					}
				}
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
			l := len(fmg3.Text)
			for i := 0; i < m; i++ {
				if isEmpty(fmg1.Text[i]) && isEmpty(fmg2.Text[i]) {
					continue
				}
				if fmg1.Text[i] != fmg2.Text[i] {
					if !isEmpty(fmg1.Text[i]) {
						if _, err = io.WriteString(of, "< "+strconv.Itoa(i)+":"+strconv.Quote(fmg1.Text[i])+"\n"); err != nil {
							return err
						}
					}
					if _, err = io.WriteString(of, "> "+strconv.Itoa(i)+":"+strconv.Quote(fmg2.Text[i])+"\n"); err != nil {
						return err
					}
					if i < l && !isEmpty(fmg3.Text[i]) {
						if _, err = io.WriteString(of, "- "+strconv.Itoa(i)+":"+strconv.Quote(fmg3.Text[i])+"\n"); err != nil {
							return err
						}
					}
					if _, err = io.WriteString(of, "= "+strconv.Itoa(i)+":\"\"\n"); err != nil {
						return err
					}
					dirty = true
				}
			}
			if len(fmg2.Text) > m {
				for i := m; i < len(fmg2.Text); i++ {
					if fmg2.Text[i] == "" || fmg2.Text[i] == "%NULL%" {
						continue
					}
					if _, err = io.WriteString(of, "> "+strconv.Itoa(i)+":"+strconv.Quote(fmg2.Text[i])+"\n"); err != nil {
						return err
					}
					if i < l {
						if _, err = io.WriteString(of, "- "+strconv.Itoa(i)+":"+strconv.Quote(fmg3.Text[i])+"\n"); err != nil {
							return err
						}
					}
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func isEmpty(s string) bool {
	return s == "" || s == "[ERROR]" || s == "%NULL%"
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
