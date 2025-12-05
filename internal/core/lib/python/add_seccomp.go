package python

import (
	"os"
	"strconv"
	"strings"
	"syscall"

	"github.com/langgenius/dify-sandbox/internal/core/lib"
	"github.com/langgenius/dify-sandbox/internal/static/python_syscall"
	"github.com/langgenius/dify-sandbox/internal/utils/log"
)

//var allow_syscalls = []int{}

func InitSeccomp(uid int, gid int, enable_network bool) error {
	log.Info("初始化 Seccomp: uid=%d, gid=%d, 启用网络=%v", uid, gid, enable_network)
	err := syscall.Chroot(".")
	if err != nil {
		log.Error("chroot 失败: %v", err)
		return err
	}
	err = syscall.Chdir("/")
	if err != nil {
		log.Error("chdir 失败: %v", err)
		return err
	}
	log.Info("chroot + chdir 成功")

	lib.SetNoNewPrivs()

	allowed_syscalls := []int{}
	allowed_not_kill_syscalls := []int{}
	allowed_not_kill_syscalls = append(allowed_not_kill_syscalls, python_syscall.ALLOW_ERROR_SYSCALLS...)

	allowed_syscall := os.Getenv("ALLOWED_SYSCALLS")
	allowed_syscall = "0,1,3,5,7,8,9,10,11,12,13,14,15,16,17,21,24,25,32,35,39,41,42,43,44,45,46,47,49,50,51,52,54,55,58,59,60,61,63,72,79,89,95,96,102,104,105,106,107,108,110,131,138,158,186,201,202,217,218,228,230,231,233,234,257,262,267,270,271,273,274,281,291,293,302,307,318,334"
	log.Info("ALLOWED_SYSCALLS: %s", allowed_syscall)
	if allowed_syscall != "" {
		nums := strings.Split(allowed_syscall, ",")
		for num := range nums {
			syscall, err := strconv.Atoi(nums[num])
			if err != nil {
				continue
			}
			allowed_syscalls = append(allowed_syscalls, syscall)
		}
		log.Info("使用环境变量 ALLOWED_SYSCALLS: 数量=%d", len(allowed_syscalls))
	} else {
		allowed_syscalls = append(allowed_syscalls, python_syscall.ALLOW_SYSCALLS...)
		if enable_network {
			allowed_syscalls = append(allowed_syscalls, python_syscall.ALLOW_NETWORK_SYSCALLS...)
		}
		log.Info("使用默认系统调用白名单: 基础=%d, 启用网络=%v", len(python_syscall.ALLOW_SYSCALLS), enable_network)
	}

	err = lib.Seccomp(allowed_syscalls, allowed_not_kill_syscalls)
	if err != nil {
		log.Error("加载 seccomp 失败: %v", err)
		return err
	}

	// setuid
	err = syscall.Setuid(uid)
	if err != nil {
		log.Error("setuid 失败: %v", err)
		return err
	}

	// setgid
	err = syscall.Setgid(gid)
	if err != nil {
		log.Error("setgid 失败: %v", err)
		return err
	}

	log.Info("Seccomp 初始化成功并完成降权")
	return nil
}
