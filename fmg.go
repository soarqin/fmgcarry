package main

import (
	"encoding/binary"
	"io"
	"os"
)

type Fmg struct {
	Filename string
	Text     []string
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
	var largestId int32 = 0
	if err = binary.Read(f, binary.LittleEndian, &header); err != nil {
		return nil, err
	}
	r := make([]FmgRange, header.RangeCount)
	if err = binary.Read(f, binary.LittleEndian, &r); err != nil {
		return nil, err
	}
	for _, rng := range r {
		if rng.Last > largestId {
			largestId = rng.Last
		}
	}
	if _, err = f.Seek(header.StringsOffset, 0); err != nil {
		return nil, err
	}
	o := make([]int64, header.StringTableSize)
	if err = binary.Read(f, binary.LittleEndian, &o); err != nil {
		return nil, err
	}
	fmg := &Fmg{filename, make([]string, largestId+1)}
	for _, rng := range r {
		for i := rng.First; i <= rng.Last; i++ {
			off := o[i-rng.First+rng.Offset]
			if off == 0 {
				fmg.Text[i] = "%NULL%"
				continue
			}
			if _, err = f.Seek(off, 0); err != nil {
				return nil, err
			}
			var s *string
			if s, err = ReadString(f); err != nil {
				return nil, err
			}
			fmg.Text[i] = *s
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
	var stSize int32 = 1
	var i int
	for i = 0; i < len(fmg.Text); i++ {
		if fmg.Text[i] != "" {
			break
		}
	}
	rng := FmgRange{
		0, int32(i), int32(i), 0,
	}
	for i = i + 1; i < len(fmg.Text); i++ {
		if fmg.Text[i] != "" {
			if int32(i) == rng.Last+1 {
				rng.Last = int32(i)
			} else {
				r = append(r, rng)
				rng = FmgRange{
					stSize, int32(i), int32(i), 0,
				}
			}
			stSize++
		}
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
			if fmg.Text[i] == "%NULL%" {
				o[i-rng.First+rng.Offset] = 0
				continue
			}
			if o[i-rng.First+rng.Offset], err = f.Seek(0, 1); err != nil {
				return err
			}
			if err = WriteString(f, fmg.Text[i]); err != nil {
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
