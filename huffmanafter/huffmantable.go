/*
Copyright (c) 2017 Simon Schmidt

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/


package huffmanafter

import "io"
import "github.com/icza/bitio"
import "github.com/icza/huffman"

type Table struct{
	T [256]*huffman.Node
	R *huffman.Node
}
func MakeTable() *Table {
	t := new(Table)
	for i := range t.T { t.T[i] = &huffman.Node{Value:huffman.ValueType(i),Count:1} }
	return t
}

func (t *Table) Incr(beg, end byte,incr int) *Table {
	for i:=beg ; i<=end ; i++ {
		t.T[i].Count += incr
	}
	return t
}

func (t *Table) IncrStr(chars string,incr int) *Table {
	for _,i := range []byte(chars) {
		t.T[i].Count += incr
	}
	return t
}

func (t *Table) Finalize() *Table {
	arr := t.T
	t.R = huffman.Build(arr[:])
	return t
}
func (t *Table) Print() {
	huffman.Print(t.R)
}

var NotOptimized = MakeTable().Finalize()

var TextOptimized = MakeTable().Incr('a','z',50).Incr('A','Z',8).Incr('0','9',40).IncrStr(".\r\n\t ",77).IncrStr("\"()+/_,!-:;<=>@",28).Finalize()


type copier interface{
	MoveLiteral() error
}

type dummy struct{
	r io.ByteReader
	w io.ByteWriter
}
func (d dummy) MoveLiteral() error {
	b,e := d.r.ReadByte()
	if e!=nil { return e }
	d.w.WriteByte(b)
	return nil
}

type compressor struct{
	r io.ByteReader
	w bitio.Writer
	t *Table
}
func (d compressor) MoveLiteral() error {
	b,e := d.r.ReadByte()
	if e!=nil { return e }
	d.w.WriteBits(d.t.T[b].Code())
	return nil
}
type decompressor struct{
	r bitio.Reader
	w io.ByteWriter
	t *Table
}
func (d decompressor) MoveLiteral() error {
	n := d.t.R
	for n.Left!=nil {
		b,e := d.r.ReadBool()
		if e!=nil { return e }
		if b { n = n.Right } else { n = n.Left }
	}
	d.w.WriteByte(byte(n.Value))
	return nil
}

