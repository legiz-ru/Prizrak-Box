//go:build windows

// Windows app-icon extraction via the Win32 API (pure syscall, no CGO — the
// Windows Wails build sets CGO_ENABLED=0). PrivateExtractIconsW pulls an icon at
// the requested size straight from the .exe; the HICON is then rendered to a
// 32-bit top-down DIB and encoded as PNG.
package services

import (
	"bytes"
	"errors"
	"image"
	"image/png"
	"syscall"
	"unsafe"
)

var (
	modUser32   = syscall.NewLazyDLL("user32.dll")
	modGdi32    = syscall.NewLazyDLL("gdi32.dll")
	modShell32x = syscall.NewLazyDLL("shell32.dll")

	procPrivateExtractIconsW = modUser32.NewProc("PrivateExtractIconsW")
	procGetIconInfo          = modUser32.NewProc("GetIconInfo")
	procDestroyIcon          = modUser32.NewProc("DestroyIcon")
	procGetDC                = modUser32.NewProc("GetDC")
	procReleaseDC            = modUser32.NewProc("ReleaseDC")

	procGetDIBits     = modGdi32.NewProc("GetDIBits")
	procDeleteObject  = modGdi32.NewProc("DeleteObject")
	procExtractIconEx = modShell32x.NewProc("ExtractIconExW")
)

type iconInfo struct {
	fIcon    int32
	xHotspot uint32
	yHotspot uint32
	hbmMask  uintptr
	hbmColor uintptr
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

const (
	biRGB        = 0
	dibRGBColors = 0
)

func fileIconPNG(path string, size int) ([]byte, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}

	var hicon uintptr
	var iconID uint32
	// PrivateExtractIconsW(path, index, cx, cy, *phicon, *piconid, nIcons, flags)
	n, _, _ := procPrivateExtractIconsW.Call(
		uintptr(unsafe.Pointer(p)), 0,
		uintptr(size), uintptr(size),
		uintptr(unsafe.Pointer(&hicon)), uintptr(unsafe.Pointer(&iconID)),
		1, 0,
	)
	if n == 0 || hicon == 0 {
		// Fallback: large (32px) associated icon via ExtractIconEx.
		var large uintptr
		cnt, _, _ := procExtractIconEx.Call(
			uintptr(unsafe.Pointer(p)), 0,
			uintptr(unsafe.Pointer(&large)), 0, 1,
		)
		if cnt == 0 || large == 0 {
			return nil, errors.New("no icon")
		}
		hicon = large
	}
	defer procDestroyIcon.Call(hicon)

	img, err := hiconToImage(hicon, size)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// hiconToImage renders an HICON's colour bitmap into an RGBA image, recovering
// the alpha channel from the AND-mask when the colour bitmap has none.
func hiconToImage(hicon uintptr, size int) (*image.NRGBA, error) {
	var ii iconInfo
	if r, _, _ := procGetIconInfo.Call(hicon, uintptr(unsafe.Pointer(&ii))); r == 0 {
		return nil, errors.New("GetIconInfo failed")
	}
	defer func() {
		if ii.hbmColor != 0 {
			procDeleteObject.Call(ii.hbmColor)
		}
		if ii.hbmMask != 0 {
			procDeleteObject.Call(ii.hbmMask)
		}
	}()

	hdc, _, _ := procGetDC.Call(0)
	if hdc == 0 {
		return nil, errors.New("GetDC failed")
	}
	defer procReleaseDC.Call(0, hdc)

	w, h := size, size
	bi := bitmapInfoHeader{
		Size:        40,
		Width:       int32(w),
		Height:      -int32(h), // top-down
		Planes:      1,
		BitCount:    32,
		Compression: biRGB,
	}

	color := make([]byte, w*h*4)
	if r, _, _ := procGetDIBits.Call(hdc, ii.hbmColor, 0, uintptr(h),
		uintptr(unsafe.Pointer(&color[0])), uintptr(unsafe.Pointer(&bi)), dibRGBColors); r == 0 {
		return nil, errors.New("GetDIBits(color) failed")
	}

	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	hasAlpha := false
	for i := 0; i < w*h; i++ {
		if color[i*4+3] != 0 {
			hasAlpha = true
			break
		}
	}

	var mask []byte
	if !hasAlpha && ii.hbmMask != 0 {
		// 1bpp AND mask: bit set => transparent.
		mbi := bitmapInfoHeader{Size: 40, Width: int32(w), Height: -int32(h), Planes: 1, BitCount: 1, Compression: biRGB}
		stride := ((w + 31) / 32) * 4
		mask = make([]byte, stride*h)
		// 1bpp DIBs need a 2-entry colour table after the header.
		buf := make([]byte, 40+8)
		copy(buf, (*[40]byte)(unsafe.Pointer(&mbi))[:])
		procGetDIBits.Call(hdc, ii.hbmMask, 0, uintptr(h),
			uintptr(unsafe.Pointer(&mask[0])), uintptr(unsafe.Pointer(&buf[0])), dibRGBColors)
		_ = stride
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			si := (y*w + x) * 4
			b, g, r, a := color[si], color[si+1], color[si+2], color[si+3]
			if !hasAlpha {
				a = 255
				if mask != nil {
					stride := ((w + 31) / 32) * 4
					bit := mask[y*stride+x/8] & (0x80 >> uint(x%8))
					if bit != 0 {
						a = 0 // masked out → transparent
					}
				}
			}
			di := img.PixOffset(x, y)
			img.Pix[di+0] = r
			img.Pix[di+1] = g
			img.Pix[di+2] = b
			img.Pix[di+3] = a
		}
	}
	return img, nil
}
