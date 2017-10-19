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

/*

This package Offers compression of already compressed data (Lz4 or Snappy) using huffman tables.

In the process of post-processing, only Literals are compressed using the static huffman tables.

In addition to that, this package can be used for any kind of Static Huffman Coding (by using the Huffman tables directly).
*/
package huffmanafter

import "io"
import "bytes"
import "github.com/icza/bitio"

func moveLz4(r io.ByteReader,w io.ByteWriter,c copier) {
	for {
		b,e := r.ReadByte(); if e!=nil { break }
		lLen := int(b >> 4)
		mLen := int(b & 0xF)
		w.WriteByte(b)
		
		// Literals
		if lLen>0 {
			if lLen == 0xF {
				b,e = r.ReadByte() ; if e!=nil { break }
				w.WriteByte(b)
				for b==0xFF {
					lLen+=0xFF
					b,e = r.ReadByte() ; if e!=nil { break }
					w.WriteByte(b)
				}
				lLen += int(b)
			}
			
			for i := 0 ; i < lLen ; i++ {
				e = c.MoveLiteral() ; if e!=nil { return ; break } // This should not happen
			}
		}
		if e!=nil { break }
		
		b,e = r.ReadByte() ; if e!=nil { break }
		w.WriteByte(b)
		b,e = r.ReadByte() ; if e!=nil { break }
		w.WriteByte(b)
		
		if mLen==0xF {
			b,e = r.ReadByte() ; if e!=nil { break }
			w.WriteByte(b)
			for b==0xFF {
				mLen+=0xFF
				b,e = r.ReadByte() ; if e!=nil { break }
				w.WriteByte(b)
			}
			mLen += int(b)
		}
	}
	return
}

func InspectLz4(src []byte) []byte {
	buf := new(bytes.Buffer)
	rdr := bytes.NewReader(src)
	moveLz4(rdr,buf,dummy{rdr,buf})
	return buf.Bytes()
}

func CompressLz4(tab *Table,src []byte) []byte {
	buf := new(bytes.Buffer)
	rdr := bytes.NewReader(src)
	bw := bitio.NewWriter(buf)
	moveLz4(rdr,bw,compressor{rdr,bw,tab})
	bw.Align()
	bw.Close()
	return buf.Bytes()
}

func DecompressLz4(tab *Table,src []byte) []byte {
	buf := new(bytes.Buffer)
	rdr := bytes.NewReader(src)
	br := bitio.NewReader(rdr)
	moveLz4(br,buf,decompressor{br,buf,tab})
	return buf.Bytes()
}


func moveSnappy(r io.ByteReader,w io.ByteWriter,c copier) {
	const (
		tagLiteral = 0x00
		tagCopy1   = 0x01
		tagCopy2   = 0x02
		tagCopy4   = 0x03
	)
	var b byte
	var e error
	
	// Skip UVarint
	for {
		b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
		if b<0x80 { break }
	}
	for {
		b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
		switch b&0x3{
		case tagLiteral:
			x := uint32(b>>2)
			switch x {
			default:
				if x<60 {
					b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
				}
			case 60:
				b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
				x  = uint32(b)
			case 61:
				b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
				x  = uint32(b)
				b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
				x |= uint32(b)<<8
			case 62:
				b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
				x  = uint32(b)
				b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
				x |= uint32(b)<<8
				b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
				x |= uint32(b)<<16
			case 63:
				b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
				x  = uint32(b)
				b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
				x |= uint32(b)<<8
				b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
				x |= uint32(b)<<16
				b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
				x |= uint32(b)<<24
			}
			length := int(x)
			for i := 0 ; i<length ; i++ {
				e = c.MoveLiteral() ; if e!=nil { return }
			}
		case tagCopy1:
			b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
		case tagCopy2:
			b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
			b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
		case tagCopy4:
			b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
			b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
			b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
			b,e = r.ReadByte() ; if e!=nil { return } ; w.WriteByte(b)
		}
	}
}

func InspectSnappy(src []byte) []byte {
	buf := new(bytes.Buffer)
	rdr := bytes.NewReader(src)
	moveSnappy(rdr,buf,dummy{rdr,buf})
	return buf.Bytes()
}

func CompressSnappy(tab *Table,src []byte) []byte {
	buf := new(bytes.Buffer)
	rdr := bytes.NewReader(src)
	bw := bitio.NewWriter(buf)
	moveSnappy(rdr,bw,compressor{rdr,bw,tab})
	bw.Align()
	bw.Close()
	return buf.Bytes()
}

func DecompressSnappy(tab *Table,src []byte) []byte {
	buf := new(bytes.Buffer)
	rdr := bytes.NewReader(src)
	br := bitio.NewReader(rdr)
	moveSnappy(br,buf,decompressor{br,buf,tab})
	return buf.Bytes()
}

