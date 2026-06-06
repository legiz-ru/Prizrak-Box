//go:build darwin

// macOS app-icon extraction via Cocoa NSWorkspace (cgo — the macOS Wails build
// uses CGO_ENABLED=1). NSWorkspace resolves the bundle icon for an executable
// path inside a .app, matching Electron's app.getFileIcon behaviour.
package services

/*
#cgo CFLAGS: -x objective-c
#cgo LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>
#include <stdlib.h>
#include <string.h>

// iconPNG returns malloc'd PNG bytes for the icon of `path` rendered at px×px,
// or NULL. The length is written to *outLen. Caller frees the buffer.
static unsigned char* iconPNG(const char* path, int px, int* outLen) {
    @autoreleasepool {
        NSString* p = [NSString stringWithUTF8String:path];
        if (p == nil) return NULL;
        NSImage* img = [[NSWorkspace sharedWorkspace] iconForFile:p];
        if (img == nil) return NULL;
        NSRect rect = NSMakeRect(0, 0, px, px);
        CGImageRef cg = [img CGImageForProposedRect:&rect context:nil hints:nil];
        if (cg == NULL) return NULL;
        NSBitmapImageRep* rep = [[NSBitmapImageRep alloc] initWithCGImage:cg];
        if (rep == nil) return NULL;
        [rep setSize:NSMakeSize(px, px)];
        NSData* data = [rep representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
        if (data == nil) return NULL;
        int len = (int)[data length];
        unsigned char* buf = (unsigned char*)malloc(len);
        if (buf == NULL) return NULL;
        memcpy(buf, [data bytes], len);
        *outLen = len;
        return buf;
    }
}
*/
import "C"

import (
	"errors"
	"unsafe"
)

func fileIconPNG(path string, size int) ([]byte, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))

	var outLen C.int
	buf := C.iconPNG(cpath, C.int(size), &outLen)
	if buf == nil || outLen <= 0 {
		return nil, errors.New("no icon")
	}
	defer C.free(unsafe.Pointer(buf))

	return C.GoBytes(unsafe.Pointer(buf), outLen), nil
}
