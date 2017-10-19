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


package lz77huff

import "github.com/icza/bitio"
import "github.com/icza/huffman"
import "github.com/maxymania/gocompress/huffmanafter"

type bincode struct{
	data uint64
	bits byte
}
func (b bincode) get() (uint64,byte) { return b.data,b.bits }

func cloneBinCode(t *[256]*huffman.Node) *[256]bincode {
	b := new([256]bincode)
	for i,n := range t {
		b[i].data,b[i].bits = n.Code()
	}
	return b
}
type histogram [256]int
func (h *histogram) scan(src []byte) {
	for i := range h { h[i] = 0 }
	for _,b := range src { h[b]++ }
}
func (h *histogram) score(b *[256]bincode) (total int) {
	for i,n := range h {
		total += int(b[i].bits)*n
	}
	return
}
type tableMeta struct{
	t      *huffmanafter.Table
	weigth *[256]bincode
	root   *huffman.Node
}
func toTableMeta(t *huffmanafter.Table) tableMeta {
	return tableMeta{
		t,
		cloneBinCode(&(t.T)),
		t.R,
	}
}

var widthTable = [...]*huffman.Node{
	{Value:5},
	{Value:8},
	{Value:10},
	{Value:12},
	{Value:16},
	{Value:20},
	{Value:24},
	{Value:31},
}
var widthTableRoot *huffman.Node


func huffRead(r bitio.Reader, n *huffman.Node) (huffman.ValueType,error) {
	for n.Left!=nil {
		b,e := r.ReadBool()
		if e!=nil { return 0,e }
		if b { n = n.Right } else { n = n.Left }
	}
	return n.Value,nil
}

func readVarLen(r bitio.Reader) (uint32,error) {
	v,e := huffRead(r,widthTableRoot) ; if e!=nil { return 0,e }
	i,e := r.ReadBits(byte(v))
	return uint32(i),e
}
func writeVarLen(w bitio.Writer, i uint32) {
	for _,n := range widthTable {
		if (1<<uint(n.Value))>i {
			w.WriteBits(n.Code())
			w.WriteBits(uint64(i),byte(n.Value))
			return
		}
	}
	panic("overflow")
}

func makeHammingTable(b bool) *huffmanafter.Table {
	tab := huffmanafter.MakeTable()
	for i := 0; i<256 ; i++ {
		j,c := i,0
		for j>0 {
			j &= j-1
			c++
		}
		if b { c = 8-c }
		tab.T[i].Count += 1<<uint(c)
	}
	tab.Finalize()
	return tab
}


var encoders = [...]tableMeta{
	toTableMeta(huffmanafter.TextOptimized),
	toTableMeta(makeHammingTable(false)),
	toTableMeta(makeHammingTable(true)),
	toTableMeta(huffmanafter.NotOptimized),
}
var encodersTable     []*huffman.Node
var encodersTableRoot   *huffman.Node

func writeHuffLiteral(w bitio.Writer,src []byte) {
	var h histogram
	ty := 0
	tsc := 0
	h.scan(src)
	
	for i,m := range encoders {
		_,bts := encodersTable[i].Code()
		sc := h.score(m.weigth)+int(bts)
		if sc<tsc || i==0 {
			ty = i
			tsc = sc
		}
	}
	w.WriteBits(encodersTable[ty].Code())
	enc := encoders[ty]
	for _,b := range src {
		w.WriteBits(enc.weigth[b].get())
	}
}
func readHuffLiteral(r bitio.Reader,dst []byte) error {
	v,e := huffRead(r,encodersTableRoot) ; if e!=nil { return e }
	dec := encoders[v]
	for i := range dst {
		v,e = huffRead(r,dec.root) ; if e!=nil { return e }
		dst[i] = byte(v)
	}
	return nil
}


func init(){
	for i := range widthTable {
		widthTable[i].Count = len(widthTable)-i
		
	}
	{ c := widthTable; widthTableRoot = huffman.Build(c[:]) }
	{
		a := make([]*huffman.Node,len(encoders))
		b := make([]*huffman.Node,len(encoders))
		for i := range encoders {
			n := &huffman.Node{Value:huffman.ValueType(i),Count:len(encoders)-i}
			a[i],b[i] = n,n
		}
		encodersTable = a
		encodersTableRoot = huffman.Build(b)
	}
}
