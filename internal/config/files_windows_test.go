//go:build windows

package config

import (
	"os"
	"path/filepath"
	"testing"
	"unsafe"

	"golang.org/x/sys/windows"
)

func assertPrivateWindowsACL(t *testing.T, path string, directory bool) {
	t.Helper()
	sd, err := windows.GetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION)
	if err != nil {
		t.Fatal("读取 Windows DACL 失败", err)
	}
	control, _, err := sd.Control()
	if err != nil || control&windows.SE_DACL_PROTECTED == 0 {
		t.Fatal("配置 ACL 必须禁止父目录权限继承", err)
	}
	dacl, _, err := sd.DACL()
	if err != nil || dacl == nil || dacl.AceCount != 1 {
		t.Fatal("配置只能有当前用户的一条访问规则", err)
	}
	var ace *windows.ACCESS_ALLOWED_ACE
	if err := windows.GetAce(dacl, 0, &ace); err != nil {
		t.Fatal(err)
	}
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		t.Fatal(err)
	}
	sid := (*windows.SID)(unsafe.Pointer(&ace.SidStart))
	if ace.Header.AceType != windows.ACCESS_ALLOWED_ACE_TYPE || !sid.Equals(user.User.Sid) {
		t.Fatal("访问规则必须只允许当前用户")
	}
	if ace.Mask&0x1f01ff != 0x1f01ff || ace.Header.AceFlags&windows.INHERITED_ACE != 0 {
		t.Fatal("当前用户应具有完整文件访问，规则不得来自继承")
	}
	inheritance := ace.Header.AceFlags & (windows.OBJECT_INHERIT_ACE | windows.CONTAINER_INHERIT_ACE)
	if directory && inheritance != windows.OBJECT_INHERIT_ACE|windows.CONTAINER_INHERIT_ACE {
		t.Fatal("家目录规则必须传递至新建子项")
	}
	if !directory && inheritance != 0 {
		t.Fatal("普通文件不应设置子目录继承标记")
	}
}

func TestWindowsCredentialACLThroughLoginUpdateLogout(t *testing.T) {
	home := isolatedHome(t)
	path, _ := ResolvePath()
	credentials, _ := CredentialsPath()
	for _, token := range []string{"sentinel-initial", "sentinel-updated"} {
		if err := SaveLogin(path, DefaultGatewayURL, token); err != nil {
			t.Fatal(err)
		}
		assertPrivateWindowsACL(t, filepath.Join(home, ".kdl"), true)
		for _, file := range []string{path, credentials, filepath.Join(home, ".kdl", ".credentials.lock")} {
			assertPrivateWindowsACL(t, file, false)
		}
	}
	if err := Logout(DefaultGatewayURL); err != nil {
		t.Fatal(err)
	}
	assertPrivateWindowsACL(t, credentials, false)
	loaded, err := Load()
	if err != nil || loaded.Token != "" {
		t.Fatal("退出应移除凭证且保留受限配置", err)
	}
}

func TestWindowsLockedConfigRollsBackCredentials(t *testing.T) {
	isolatedHome(t)
	path, _ := ResolvePath()
	if err := SaveLogin(path, DefaultGatewayURL, "sentinel-original"); err != nil {
		t.Fatal(err)
	}
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	// 拒绝 FILE_SHARE_DELETE，使原子替换失败；读取仍允许，不需要管理员或额外账号。
	handle, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil,
		windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer windows.CloseHandle(handle)
	if err := SaveLogin(path, DefaultGatewayURL, "sentinel-new"); err == nil {
		t.Fatal("配置被占用时应失败")
	}
	loaded, err := Load()
	if err != nil || loaded.Token != "sentinel-original" {
		t.Fatal("配置替换失败后必须恢复原凭证", err)
	}
	credentials, _ := CredentialsPath()
	assertPrivateWindowsACL(t, credentials, false)
	files, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".kdl-write-*"))
	if len(files) != 0 {
		t.Fatal("失败后不应遗留凭证临时文件")
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("原普通配置必须保留", err)
	}
}
