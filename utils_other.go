// +build !linux,!darwin

package ssf

func trylockFile(path string) error {
	return nil
}
