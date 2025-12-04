package python

import (
	"crypto/rand"
	_ "embed"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/langgenius/dify-sandbox/internal/utils/log"

	"github.com/google/uuid"
	"github.com/langgenius/dify-sandbox/internal/core/runner"
	"github.com/langgenius/dify-sandbox/internal/core/runner/types"
	"github.com/langgenius/dify-sandbox/internal/static"
)

type PythonRunner struct {
	runner.TempDirRunner
}

//go:embed prescript.py
var sandbox_fs []byte

func (p *PythonRunner) Run(
	code string,
	timeout time.Duration,
	stdin []byte,
	preload string,
	options *types.RunnerOptions,
) (chan []byte, chan []byte, chan bool, error) {
	configuration := static.GetDifySandboxGlobalConfigurations()

	// initialize the environment
	untrusted_code_path, key, err := p.InitializeEnvironment(code, preload, options)
	if err != nil {
		log.Error("InitializeEnvironment 失败: %v", err)
		return nil, nil, nil, err
	}
	log.Info("初始化完成: 脚本=%s", untrusted_code_path)

	// capture the output
	output_handler := runner.NewOutputCaptureRunner()
	output_handler.SetTimeout(timeout)
	output_handler.SetAfterExitHook(func() {
		// remove untrusted code
		os.Remove(untrusted_code_path)
	})

	// create a new process
	workdir := ""
	if options != nil && options.TaskID != "" {
		workdir = fmt.Sprintf("/tmp/%s", options.TaskID)
	}
	log.Info("执行配置: 工作目录=%s, 库路径=%s, 启用网络=%v", workdir, LIB_PATH, options != nil && options.EnableNetwork)
	cmd := exec.Command(
		configuration.PythonPath,
		untrusted_code_path,
		LIB_PATH,
		key,
		workdir,
	)
	cmd.Env = []string{}
	cmd.Dir = LIB_PATH
	log.Debug("将要执行命令: path=%s, dir=%s, args=%v", cmd.Path, cmd.Dir, cmd.Args)

	if configuration.Proxy.Socks5 != "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("HTTPS_PROXY=%s", configuration.Proxy.Socks5))
		cmd.Env = append(cmd.Env, fmt.Sprintf("HTTP_PROXY=%s", configuration.Proxy.Socks5))
	} else if configuration.Proxy.Https != "" || configuration.Proxy.Http != "" {
		if configuration.Proxy.Https != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("HTTPS_PROXY=%s", configuration.Proxy.Https))
		}
		if configuration.Proxy.Http != "" {
			cmd.Env = append(cmd.Env, fmt.Sprintf("HTTP_PROXY=%s", configuration.Proxy.Http))
		}
	}

	if len(configuration.AllowedSyscalls) > 0 {
		cmd.Env = append(cmd.Env,
			fmt.Sprintf("ALLOWED_SYSCALLS=%s",
				strings.Trim(strings.Join(strings.Fields(fmt.Sprint(configuration.AllowedSyscalls)), ","), "[]"),
			),
		)
		log.Info("启用自定义 ALLOWED_SYSCALLS 覆盖: 数量=%d", len(configuration.AllowedSyscalls))
	}

	err = output_handler.CaptureOutput(cmd)
	if err != nil {
		log.Error("捕获输出失败: %v", err)
		return nil, nil, nil, err
	}

	return output_handler.GetStdout(), output_handler.GetStderr(), output_handler.GetDone(), nil
}

func (p *PythonRunner) InitializeEnvironment(code string, preload string, options *types.RunnerOptions) (string, string, error) {
	if !checkLibAvaliable() {
		// ensure environment is reversed
		releaseLibBinary(false)
	}

	// create a tmp dir and copy the python script
	temp_code_name := strings.ReplaceAll(uuid.New().String(), "-", "_")
	temp_code_name = strings.ReplaceAll(temp_code_name, "/", ".")

	script := strings.Replace(
		string(sandbox_fs),
		"{{uid}}", strconv.Itoa(static.SANDBOX_USER_UID), 1,
	)

	script = strings.Replace(
		script,
		"{{gid}}", strconv.Itoa(static.SANDBOX_GROUP_ID), 1,
	)

	if options.EnableNetwork {
		script = strings.Replace(
			script,
			"{{enable_network}}", "1", 1,
		)
	} else {
		script = strings.Replace(
			script,
			"{{enable_network}}", "0", 1,
		)
	}

	script = strings.Replace(
		script,
		"{{preload}}",
		fmt.Sprintf("%s\n", preload),
		1,
	)

	// generate a random 512 bit key
	key_len := 64
	key := make([]byte, key_len)
	_, err := rand.Read(key)
	if err != nil {
		return "", "", err
	}

	// encrypt the code
	encrypted_code := make([]byte, len(code))
	for i := 0; i < len(code); i++ {
		encrypted_code[i] = code[i] ^ key[i%key_len]
	}

	// encode code using base64
	code = base64.StdEncoding.EncodeToString(encrypted_code)
	// encode key using base64
	encoded_key := base64.StdEncoding.EncodeToString(key)

	code = strings.Replace(
		script,
		"{{code}}",
		code,
		1,
	)

	untrusted_code_path := fmt.Sprintf("%s/tmp/%s.py", LIB_PATH, temp_code_name)
	err = os.MkdirAll(path.Dir(untrusted_code_path), 0755)
	if err != nil {
		return "", "", err
	}
	err = os.WriteFile(untrusted_code_path, []byte(code), 0755)
	if err != nil {
		log.Error("写入脚本失败: %v", err)
		return "", "", err
	}

	// ensure per-task workdir exists and is owned by sandbox user
	if options != nil && options.TaskID != "" {
		workdir := path.Join(LIB_PATH, "tmp", options.TaskID)
		_ = os.MkdirAll(workdir, 0770)
		_ = os.Chown(workdir, static.SANDBOX_USER_UID, static.SANDBOX_GROUP_ID)
		log.Info("已准备工作目录: %s, uid=%d, gid=%d", workdir, static.SANDBOX_USER_UID, static.SANDBOX_GROUP_ID)
	}

	return untrusted_code_path, encoded_key, nil
}
