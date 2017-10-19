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

// LZ77-like Block coder with Huffman encoding.
package lz77huff

import "encoding/binary"
import "fmt"
import "bytes"
import "github.com/icza/bitio"

func safelen(a, b int) int {
	if a>b { return b }
	return a
}
func commonPrefix(a, b []byte) int {
	l := safelen(len(a),len(b))
	for i := 0 ; i<l ; i++ {
		if a[i]!=b[i] { return i }
	}
	return l
}

// This function does nonsense!!
func WalkBuffer(src []byte){

	hashtab := make(map[uint32]int,len(src)/4)
	var begin,end int
	lng := len(src)
	
	if lng>=4{
		hashtab[binary.LittleEndian.Uint32(src)] = 0
	}
	
	lst := lng-4
	for end=4 ; end<lst ; end++ {
		i := end-4
		hashtab[binary.LittleEndian.Uint32(src[i:])] = i
		ke := binary.LittleEndian.Uint32(src[end:])
		i,ok := hashtab[ke]
		if ok {
			fmt.Println(begin,end,lng)
			cp := commonPrefix(src[i:end],src[end:])
			fmt.Println("L",begin,end)
			fmt.Println("S",i,i+cp)
			begin = end+cp
			end = begin
		}
	}
	fmt.Println("L",begin,lng)
}

func writerSubst(w bitio.Writer,i int,l int) {
	w.WriteBool(true)
	w.WriteBool(true)
	writeVarLen(w,uint32(i))
	writeVarLen(w,uint32(l))
}
func writerLiteral(w bitio.Writer,data []byte) {
	w.WriteBool(false)
	writeVarLen(w,uint32(len(data)))
	writeHuffLiteral(w,data)
	//w.Write(data)
}

func expand(s []byte,n int) []byte {
	if cap(s)<n {
		v := make([]byte,n)
		copy(s,v)
		return v
	}
	return s[:n]
}

func Compress(src []byte) []byte {
	buf := new(bytes.Buffer)
	w := bitio.NewWriter(buf)
	{
		var buf [16]byte
		i := binary.PutUvarint(buf[:],uint64(len(src)))
		w.Write(buf[:i])
	}
	
	hashtab := make(map[uint32]int,len(src)/4)
	var begin,end int
	lng := len(src)
	
	if lng>=4{
		hashtab[binary.LittleEndian.Uint32(src)] = 0
	}
	
	lst := lng-4
	for end=4 ; end<lst ; end++ {
		i := end-4
		hashtab[binary.LittleEndian.Uint32(src[i:])] = i
		ke := binary.LittleEndian.Uint32(src[end:])
		i,ok := hashtab[ke]
		if ok {
			cp := commonPrefix(src[i:end],src[end:])
			writerLiteral(w,src[begin:end])
			writerSubst(w,i,cp)
			begin = end+cp
			end = begin
		}
	}
	writerLiteral(w,src[begin:])
	w.WriteBool(true)
	w.WriteBool(false)
	
	w.Align()
	w.Close()
	
	return buf.Bytes()
}

func Uncompress(src []byte) []byte {
	r := bitio.NewReader(bytes.NewReader(src))
	lnghdr,e := binary.ReadUvarint(r)
	if e!=nil { return nil }
	if lnghdr>(1<<24) {
		lnghdr = (1<<24)
	}
	dst := make([]byte,0,int(lnghdr))
	
	for {
		t,e := r.ReadBool() ; if e!=nil { break }
		if t {
			t,e = r.ReadBool() ; if e!=nil { break }
			if !t { break }
			pos,e := readVarLen(r) ; if e!=nil { break }
			l,e := readVarLen(r) ; if e!=nil { break }
			if int(pos+l)>len(dst) { return nil } // Error
			dst = append(dst,dst[pos:pos+l]...)
		} else {
			l,e := readVarLen(r) ; if e!=nil { break }
			pos := len(dst)
			dst = expand(dst,pos+int(l))
			e = readHuffLiteral(r,dst[pos:]) ; if e!=nil { break }
		}
	}
	return dst
}

