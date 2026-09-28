//go:build windows

package extract

import "testing"

func TestMsiexecCmdLineQuotesPropertyValue(t *testing.T) {
	got := msiexecCmdLine(`C:\Windows\System32\msiexec.exe`, `C:\Users\a b\LibreOffice.msi`, `C:\Users\a b\.lovm\install`, `C:\Temp\x.log`)
	want := `"C:\Windows\System32\msiexec.exe" /a "C:\Users\a b\LibreOffice.msi" /qn TARGETDIR="C:\Users\a b\.lovm\install" /l* "C:\Temp\x.log"`
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}
