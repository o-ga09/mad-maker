package media

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func writePNG(t *testing.T, path string) {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 3))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, buf.String())
}

func writeJPEG(t *testing.T, path string) {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 4, 5)), nil); err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, buf.String())
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestListImages(t *testing.T) {
	tests := []struct {
		name  string
		files []string // 作成するファイル名（"/" 終わりはフォルダ）
		want  []string
	}{
		{
			name:  "自然順に並ぶ",
			files: []string{"img10.png", "img2.png", "img1.png"},
			want:  []string{"img1.png", "img2.png", "img10.png"},
		},
		{
			name:  "拡張子は大文字小文字を区別しない",
			files: []string{"a.PNG", "b.Jpg", "c.JPEG", "d.jpeg"},
			want:  []string{"a.PNG", "b.Jpg", "c.JPEG", "d.jpeg"},
		},
		{
			name:  "対象外のファイルとサブフォルダは無視する",
			files: []string{"a.png", "b.gif", "c.txt", "sub/", "sub.png/", "noext"},
			want:  []string{"a.png"},
		},
		{
			name:  "ファイル名の大文字小文字を無視して並べる",
			files: []string{"B.png", "a.png"},
			want:  []string{"a.png", "B.png"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, f := range tt.files {
				p := filepath.Join(dir, f)
				if strings.HasSuffix(f, "/") {
					if err := os.Mkdir(p, 0o750); err != nil {
						t.Fatal(err)
					}
					continue
				}
				writeFile(t, p, "x")
			}
			got, err := ListImages(dir)
			if err != nil {
				t.Fatalf("ListImages() error = %v", err)
			}
			var want []string
			for _, w := range tt.want {
				want = append(want, filepath.Join(dir, w))
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("ListImages() = %v, want %v", got, want)
			}
		})
	}
}

func TestListImagesErrors(t *testing.T) {
	t.Run("画像が 1 枚もない", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, filepath.Join(dir, "a.txt"), "x")
		if _, err := ListImages(dir); err == nil {
			t.Fatal("error = nil, want error")
		}
	})
	t.Run("フォルダが存在しない", func(t *testing.T) {
		if _, err := ListImages(filepath.Join(t.TempDir(), "none")); err == nil {
			t.Fatal("error = nil, want error")
		}
	})
}

func TestDecode(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "a.png")
	jpgPath := filepath.Join(dir, "b.JPG")
	writePNG(t, pngPath)
	writeJPEG(t, jpgPath)

	tests := []struct {
		name string
		path string
		want image.Point
	}{
		{"png", pngPath, image.Pt(2, 3)},
		{"jpeg", jpgPath, image.Pt(4, 5)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			img, err := Decode(tt.path)
			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			if got := img.Bounds().Size(); got != tt.want {
				t.Errorf("size = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDecodeErrors(t *testing.T) {
	dir := t.TempDir()
	broken := filepath.Join(dir, "broken.png")
	writeFile(t, broken, "not an image")

	_, err := Decode(broken)
	if err == nil {
		t.Fatal("error = nil, want error")
	}
	if !strings.Contains(err.Error(), "broken.png") {
		t.Errorf("error %q does not contain file name", err)
	}

	if _, err := Decode(filepath.Join(dir, "missing.png")); err == nil {
		t.Fatal("missing file: error = nil, want error")
	}
}

func TestLoadImages(t *testing.T) {
	t.Run("すべて読み込める", func(t *testing.T) {
		dir := t.TempDir()
		writePNG(t, filepath.Join(dir, "img2.png"))
		writeJPEG(t, filepath.Join(dir, "img1.jpg"))
		imgs, err := LoadImages(dir)
		if err != nil {
			t.Fatalf("LoadImages() error = %v", err)
		}
		if len(imgs) != 2 {
			t.Fatalf("len = %d, want 2", len(imgs))
		}
		if got := imgs[0].Bounds().Size(); got != image.Pt(4, 5) {
			t.Errorf("imgs[0] size = %v, want jpg (4,5)", got)
		}
	})
	t.Run("壊れた画像のファイル名がエラーに含まれる", func(t *testing.T) {
		dir := t.TempDir()
		writePNG(t, filepath.Join(dir, "ok.png"))
		writeFile(t, filepath.Join(dir, "bad.jpg"), "garbage")
		_, err := LoadImages(dir)
		if err == nil {
			t.Fatal("error = nil, want error")
		}
		if !strings.Contains(err.Error(), "bad.jpg") {
			t.Errorf("error %q does not contain bad.jpg", err)
		}
	})
	t.Run("画像がない", func(t *testing.T) {
		if _, err := LoadImages(t.TempDir()); err == nil {
			t.Fatal("error = nil, want error")
		}
	})
}

func TestNaturalLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"img2.png", "img10.png", true},
		{"img10.png", "img2.png", false},
		{"a.png", "b.png", true},
		{"img1.png", "img1.png", false},
		{"img01.png", "img1.png", false},
		{"img1.png", "img01.png", true},
	}
	for _, tt := range tests {
		if got := naturalLess(tt.a, tt.b); got != tt.want {
			t.Errorf("naturalLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
