package main

import (
	"os"
	"path/filepath"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	if len(os.Args) < 4 {
		println("Usage: fmgcarry <fmg1> <fmg2> <fmg3>")
		return
	}
	var fmg1, fmg2, fmg3 *Fmg
	var err error
	if err = filepath.WalkDir(os.Args[1], func(path string, d os.DirEntry, err error) error {
		if d.IsDir() || filepath.Ext(path) != ".fmg" {
			return nil
		}
		print("Processing ", path, "...")
		if fmg1, err = FmgLoad(path); err != nil {
			return err
		}
		if fmg2, err = FmgLoad(filepath.Join(os.Args[2], d.Name())); err != nil {
			return err
		}
		if fmg3, err = FmgLoad(filepath.Join(os.Args[3], d.Name())); err != nil {
			return err
		}
		m := min(len(fmg1.Text), len(fmg2.Text))
		dirty := false
		if len(fmg3.Text) < len(fmg2.Text) {
			fmg3.Text = append(fmg3.Text, make([]string, len(fmg2.Text)-len(fmg3.Text))...)
			copy(fmg3.Text[m:], fmg2.Text[m:])
			dirty = true
		}
		for i := 0; i < m; i++ {
			if fmg1.Text[i] != fmg2.Text[i] {
				fmg3.Text[i] = fmg2.Text[i]
				dirty = true
			}
		}
		if dirty {
			if err = fmg3.Save(); err != nil {
				return err
			}
		}
		println("Done")
		return nil
	}); err != nil {
		panic(err)
	}
}
