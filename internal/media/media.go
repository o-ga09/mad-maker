// Package media は画像フォルダの走査とデコードを行う。
package media

import (
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ErrNoImages は対象の画像が 1 枚もないときに返る。
var ErrNoImages = errors.New("no images found")

// ListImages は dir 直下の .png / .jpg / .jpeg（大文字小文字を区別しない）を
// ファイル名の自然順で返す。サブフォルダと他の拡張子は無視する。
// 返すパスは dir を先頭に持つ。
func ListImages(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read image dir %q: %w", dir, err)
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() || !isImageName(e.Name()) {
			continue
		}
		names = append(names, e.Name())
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("list images in %q: %w", dir, ErrNoImages)
	}
	sort.SliceStable(names, func(i, j int) bool { return naturalLess(names[i], names[j]) })
	paths := make([]string, len(names))
	for i, n := range names {
		paths[i] = filepath.Join(dir, n)
	}
	return paths, nil
}

func isImageName(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".png", ".jpg", ".jpeg":
		return true
	}
	return false
}

// Decode は path の画像ファイルをデコードする。形式は拡張子で判断する。
func Decode(path string) (img image.Image, err error) {
	root, err := os.OpenRoot(filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("open image %q: %w", path, err)
	}
	defer func() {
		if cerr := root.Close(); cerr != nil && err == nil {
			img, err = nil, fmt.Errorf("close image dir of %q: %w", path, cerr)
		}
	}()
	f, err := root.Open(filepath.Base(path))
	if err != nil {
		return nil, fmt.Errorf("open image %q: %w", path, err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			img, err = nil, fmt.Errorf("close image %q: %w", path, cerr)
		}
	}()

	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		img, err = png.Decode(f)
	case ".jpg", ".jpeg":
		img, err = jpeg.Decode(f)
	default:
		return nil, fmt.Errorf("decode image %q: unsupported extension", path)
	}
	if err != nil {
		return nil, fmt.Errorf("decode image %q: %w", path, err)
	}
	return img, nil
}

// LoadImages は dir の画像を自然順にすべてデコードして返す。
// 壊れた画像があれば、そのファイル名を含むエラーを返す。
func LoadImages(dir string) ([]image.Image, error) {
	paths, err := ListImages(dir)
	if err != nil {
		return nil, err
	}
	imgs := make([]image.Image, 0, len(paths))
	for _, p := range paths {
		img, err := Decode(p)
		if err != nil {
			return nil, err
		}
		imgs = append(imgs, img)
	}
	return imgs, nil
}

// naturalLess は数字の並びを数値として比較する自然順の a < b を返す。
// 英字は大文字小文字を区別せず比較し、同順位なら数字の桁数が少ない方を前にする。
func naturalLess(a, b string) bool {
	la, lb := strings.ToLower(a), strings.ToLower(b)
	i, j := 0, 0
	for i < len(la) && j < len(lb) {
		if isDigit(la[i]) && isDigit(lb[j]) {
			si := i
			for i < len(la) && isDigit(la[i]) {
				i++
			}
			sj := j
			for j < len(lb) && isDigit(lb[j]) {
				j++
			}
			na := strings.TrimLeft(la[si:i], "0")
			nb := strings.TrimLeft(lb[sj:j], "0")
			if len(na) != len(nb) {
				return len(na) < len(nb)
			}
			if na != nb {
				return na < nb
			}
			if i-si != j-sj {
				return i-si < j-sj
			}
			continue
		}
		if la[i] != lb[j] {
			return la[i] < lb[j]
		}
		i++
		j++
	}
	if len(la)-i != len(lb)-j {
		return len(la)-i < len(lb)-j
	}
	return a < b
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
