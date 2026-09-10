package config

import (
	"fmt"

	"golang.org/x/sys/windows"
)

func restrictAccess(path string, directory bool) error {
	user, err := windows.GetCurrentProcessToken().GetTokenUser()
	if err != nil {
		return fmt.Errorf("读取当前用户 SID 失败；请检查 Windows 用户会话")
	}
	flags := ""
	if directory {
		flags = "OICI"
	}
	sd, err := windows.SecurityDescriptorFromString("D:P(A;" + flags + ";FA;;;" + user.User.Sid.String() + ")")
	if err != nil {
		return fmt.Errorf("创建用户访问规则失败；请检查 Windows 权限配置")
	}
	dacl, _, err := sd.DACL()
	if err == nil {
		err = windows.SetNamedSecurityInfo(path, windows.SE_FILE_OBJECT, windows.DACL_SECURITY_INFORMATION|windows.PROTECTED_DACL_SECURITY_INFORMATION, nil, nil, dacl, nil)
	}
	if err != nil {
		return fmt.Errorf("设置配置文件 %s 的用户权限失败；请检查文件系统 ACL 支持", path)
	}
	return nil
}

func replaceFile(from, to string) error {
	src, err := windows.UTF16PtrFromString(from)
	if err != nil {
		return err
	}
	dst, err := windows.UTF16PtrFromString(to)
	if err != nil {
		return err
	}
	return windows.MoveFileEx(src, dst, windows.MOVEFILE_REPLACE_EXISTING|windows.MOVEFILE_WRITE_THROUGH)
}
