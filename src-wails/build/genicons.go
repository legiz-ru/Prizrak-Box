//go:build ignore

// Generates the Windows/Linux icon assets from the app icon
// (src-wails/build/appicon.png) so the tray, taskbar and .exe all show the app
// icon, crisp at small sizes. Plain Go (no external tools), high-quality
// area-averaging downscale, PNG-in-ICO encoding.
//
// Run from the src-wails directory:
//
//	go run build/genicons.go
//
// Produces (all under src-wails/build):
//
//	tray.ico     multi-size Windows tray icon (16,20,24,32,40,48,64,128,256)
//	tray.png     256px Linux tray icon (= app icon)
//	appicon.ico  multi-size icon embedded into Prizrak-Box.exe via go-winres
package main

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
)

func main() {
	src := loadPNG("build/appicon.png")

	sizes := []int{256, 128, 64, 48, 40, 32, 24, 20, 16}
	imgs := make([]*image.NRGBA, 0, len(sizes))
	for _, s := range sizes {
		imgs = append(imgs, resizeArea(src, s))
	}

	writeICO("build/tray.ico", imgs)
	writeICO("build/appicon.ico", imgs)
	writePNG("build/tray.png", resizeArea(src, 256))
	log.Println("icons generated: build/tray.ico, build/tray.png, build/appicon.ico")
}

func loadPNG(p string) *image.NRGBA {
	f, err := os.Open(p)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	im, err := png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	b := im.Bounds()
	n := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(n, n.Bounds(), im, b.Min, draw.Src)
	return n
}

// resizeArea downscales to size×size by averaging each destination pixel over
// its source box, alpha-premultiplied so transparent edges don't bleed.
func resizeArea(src *image.NRGBA, size int) *image.NRGBA {
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	for y := 0; y < size; y++ {
		sy0 := y * sh / size
		sy1 := (y + 1) * sh / size
		if sy1 <= sy0 {
			sy1 = sy0 + 1
		}
		for x := 0; x < size; x++ {
			sx0 := x * sw / size
			sx1 := (x + 1) * sw / size
			if sx1 <= sx0 {
				sx1 = sx0 + 1
			}
			var sumR, sumG, sumB, sumA float64
			var n float64
			for sy := sy0; sy < sy1; sy++ {
				for sx := sx0; sx < sx1; sx++ {
					c := src.NRGBAAt(sx, sy)
					a := float64(c.A) / 255
					sumR += float64(c.R) * a
					sumG += float64(c.G) * a
					sumB += float64(c.B) * a
					sumA += float64(c.A)
					n++
				}
			}
			avgA := sumA / n
			px := dst.PixOffset(x, y)
			if avgA <= 0 {
				dst.Pix[px+0] = 0
				dst.Pix[px+1] = 0
				dst.Pix[px+2] = 0
				dst.Pix[px+3] = 0
				continue
			}
			// Un-premultiply: straight = premultAvg / (avgA/255)
			scale := 255 / (n * avgA)
			dst.Pix[px+0] = clamp8(sumR * scale)
			dst.Pix[px+1] = clamp8(sumG * scale)
			dst.Pix[px+2] = clamp8(sumB * scale)
			dst.Pix[px+3] = clamp8(avgA)
		}
	}
	return dst
}

func clamp8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func writePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
}

// writeICO writes a multi-image .ico with each entry stored as PNG (supported by
// Windows Vista+, the same encoding the previous tray.ico used).
func writeICO(path string, imgs []*image.NRGBA) {
	type entry struct {
		w, h int
		data []byte
	}
	entries := make([]entry, 0, len(imgs))
	for _, im := range imgs {
		entries = append(entries, entry{im.Bounds().Dx(), im.Bounds().Dy(), encodePNG(im)})
	}

	out, err := os.Create(path)
	if err != nil {
		log.Fatal(err)
	}
	defer out.Close()

	// ICONDIR
	hdr := make([]byte, 6)
	binary.LittleEndian.PutUint16(hdr[0:], 0) // reserved
	binary.LittleEndian.PutUint16(hdr[2:], 1) // type 1 = icon
	binary.LittleEndian.PutUint16(hdr[4:], uint16(len(entries)))
	out.Write(hdr)

	offset := 6 + 16*len(entries)
	for _, e := range entries {
		ent := make([]byte, 16)
		ent[0] = byte(e.w & 0xff) // 0 means 256
		ent[1] = byte(e.h & 0xff)
		ent[2] = 0                                 // color count
		ent[3] = 0                                 // reserved
		binary.LittleEndian.PutUint16(ent[4:], 1)  // planes
		binary.LittleEndian.PutUint16(ent[6:], 32) // bit count
		binary.LittleEndian.PutUint32(ent[8:], uint32(len(e.data)))
		binary.LittleEndian.PutUint32(ent[12:], uint32(offset))
		out.Write(ent)
		offset += len(e.data)
	}
	for _, e := range entries {
		out.Write(e.data)
	}
}

func encodePNG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		log.Fatal(err)
	}
	return buf.Bytes()
}
