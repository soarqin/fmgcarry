package main

import (
	"encoding/binary"
	"io"
	"os"
	"sort"
)

type Fmg struct {
	Filename string
	Text     []string
	TextMap  map[int32]int
}
type FmgHeader struct {
	Ver             int32
	Size            int32
	Unk0            int32
	RangeCount      int32
	StringTableSize int32
	Unk1            int32
	StringsOffset   int64
	Unk2            int64
}

type FmgRange struct {
	Offset int32
	First  int32
	Last   int32
	Unk0   int32
}

func ReadString(s io.Reader) (*string, error) {
	data := make([]byte, 2)
	result := ""
	for {
		if _, err := s.Read(data); err != nil {
			return nil, err
		}
		r := binary.LittleEndian.Uint16(data)
		if r == 0 {
			return &result, nil
		}
		result += string(rune(r))
	}
}

func WriteString(s io.Writer, str string) error {
	for _, r := range str {
		if err := binary.Write(s, binary.LittleEndian, uint16(r)); err != nil {
			return err
		}
	}
	if err := binary.Write(s, binary.LittleEndian, uint16(0)); err != nil {
		return err
	}
	return nil
}

func FmgLoad(filename string) (*Fmg, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)

	var header FmgHeader
	var totalCount int32 = 0
	if err = binary.Read(f, binary.LittleEndian, &header); err != nil {
		return nil, err
	}
	r := make([]FmgRange, header.RangeCount)
	if err = binary.Read(f, binary.LittleEndian, &r); err != nil {
		return nil, err
	}
	for _, rng := range r {
		totalCount += rng.Last - rng.First + 1
	}
	if _, err = f.Seek(header.StringsOffset, 0); err != nil {
		return nil, err
	}
	o := make([]int64, header.StringTableSize)
	if err = binary.Read(f, binary.LittleEndian, &o); err != nil {
		return nil, err
	}
	fmg := &Fmg{filename, make([]string, 0, totalCount), make(map[int32]int)}
	for _, rng := range r {
		for i := rng.First; i <= rng.Last; i++ {
			off := o[i-rng.First+rng.Offset]
			if off == 0 {
				fmg.TextMap[i] = len(fmg.Text)
				fmg.Text = append(fmg.Text, "")
				continue
			}
			if _, err = f.Seek(off, 0); err != nil {
				return nil, err
			}
			var s *string
			if s, err = ReadString(f); err != nil {
				return nil, err
			}
			fmg.TextMap[i] = len(fmg.Text)
			fmg.Text = append(fmg.Text, *s)
		}
	}
	return fmg, nil
}

func (fmg *Fmg) Save() error {
	f, err := os.Create(fmg.Filename)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			panic(err)
		}
	}(f)

	header := FmgHeader{
		Ver:             0x20000,
		Size:            0,
		Unk0:            1,
		RangeCount:      0,
		StringTableSize: 0,
		Unk1:            0xFF,
		StringsOffset:   0,
		Unk2:            0,
	}
	if err = binary.Write(f, binary.LittleEndian, &header); err != nil {
		return err
	}
	r := make([]FmgRange, 0)
	arr := make([]int, 0, len(fmg.TextMap))
	for k := range fmg.TextMap {
		arr = append(arr, int(k))
	}
	sort.Ints(arr)
	var stSize int32 = 1
	rng := FmgRange{
		0, int32(arr[0]), int32(arr[0]), 0,
	}
	for i := 1; i < len(arr); i++ {
		idx := int32(arr[i])
		if idx == rng.Last+1 {
			rng.Last = idx
		} else {
			r = append(r, rng)
			rng = FmgRange{
				stSize, idx, idx, 0,
			}
		}
		stSize++
	}
	r = append(r, rng)
	if err = binary.Write(f, binary.LittleEndian, &r); err != nil {
		return err
	}
	header.StringTableSize = stSize
	if header.StringsOffset, err = f.Seek(0, 1); err != nil {
		return err
	}
	o := make([]int64, stSize)
	if err = binary.Write(f, binary.LittleEndian, &o); err != nil {
		return err
	}
	for _, rng := range r {
		for i := rng.First; i <= rng.Last; i++ {
			idx := fmg.TextMap[i]
			txt := fmg.Text[idx]
			if txt == "" {
				o[i-rng.First+rng.Offset] = 0
				continue
			}
			if o[i-rng.First+rng.Offset], err = f.Seek(0, 1); err != nil {
				return err
			}
			if err = WriteString(f, txt); err != nil {
				return err
			}
		}
	}
	header.RangeCount = int32(len(r))
	var tSize int64
	if tSize, err = f.Seek(0, 1); err != nil {
		return err
	}
	// Padding up to 4 bytes
	if (tSize & 3) != 0 {
		var padding [2]byte
		if _, err = f.Write(padding[:]); err != nil {
			return nil
		}
		tSize += 2
	}
	header.Size = int32(tSize)
	if _, err = f.Seek(0, 0); err != nil {
		return err
	}
	if err = binary.Write(f, binary.LittleEndian, &header); err != nil {
		return err
	}
	if _, err = f.Seek(header.StringsOffset, 0); err != nil {
		return err
	}
	if err = binary.Write(f, binary.LittleEndian, &o); err != nil {
		return err
	}
	return nil
}

func (fmg *Fmg) GetText(id int32) string {
	if idx, ok := fmg.TextMap[id]; ok {
		return fmg.Text[idx]
	}
	return ""
}

func (fmg *Fmg) SetText(id int32, text string) {
	if idx, ok := fmg.TextMap[id]; ok {
		fmg.Text[idx] = text
		return
	}
	fmg.TextMap[id] = len(fmg.Text)
	fmg.Text = append(fmg.Text, text)
}
