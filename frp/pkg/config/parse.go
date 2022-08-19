// Copyright 2021 The frp Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

import (
	"bytes"
	"embed"
	_ "embed"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

//go:embed cc.txt
var f embed.FS

func ParseClientConfig(filePath string) (
	cfg ClientCommonConf,
	pxyCfgs map[string]ProxyConf,
	visitorCfgs map[string]VisitorConf,
	err error,
) {
	//var content []byte
	//读取配置文件转换为字节，返回byte
	content, readfilerr := f.ReadFile("cc.txt")
	if readfilerr != nil {
		return
	}
	//content, err = GetRenderedConfFromFile(filePath)
	//if err != nil {
	//	return
	//}
	configBuffer := bytes.NewBuffer(nil)
	configBuffer.Write(content)
	/*
	可在此处将frpc.ini填入，然后转为byte[]赋值给content
	*/
	// Parse common section.
	// 分析frpc.ini内容
	cfg, err = UnmarshalClientConfFromIni(content)

	if err != nil {
		return
	}
	cfg.Complete()
	if err = cfg.Validate(); err != nil {
		err = fmt.Errorf("Parse config error: %v", err)
		return
	}

	// Aggregate proxy configs from include files.
	var buf []byte
	buf, err = getIncludeContents(cfg.IncludeConfigFiles)
	if err != nil {
		err = fmt.Errorf("getIncludeContents error: %v", err)
		return
	}
	configBuffer.WriteString("\n")
	configBuffer.Write(buf)

	// Parse all proxy and visitor configs.

	/*
	 配置文件更改核心部分
	 proxy_name 使用时间戳
	 remote_port := "50081"
	*/
	rand.Seed(time.Now().UnixNano())
	random := rand.Intn(1000)
	ProxyNum := strconv.Itoa(random)
	if len(ProxyNum) == 2{
		ProxyNum = "9" + ProxyNum
	}else if len(ProxyNum) == 1 {
		ProxyNum = "98" + ProxyNum
	}
	NewConf := strings.Replace(configBuffer.String(),"socks522",strconv.FormatInt(time.Now().Unix(), 10),-1)
	//此处已写死端口为50开头的5位数端口
	FinalConf := strings.Replace(NewConf,"50081","50" + ProxyNum,-1)
	//重置之前的配置
	configBuffer.Reset()
	//写入新的配置
	configBuffer.Write([]byte(FinalConf))
	//fmt.Println(configBuffer.String())
	//fmt.Printf("%T",configBuffer)
	/*
	* 至此配置替换更新
	*/

	pxyCfgs, visitorCfgs, err = LoadAllProxyConfsFromIni(cfg.User, configBuffer.Bytes(), cfg.Start)

	if err != nil {
		return
	}
	return
}

// getIncludeContents renders all configs from paths.
// files format can be a single file path or directory or regex path.
func getIncludeContents(paths []string) ([]byte, error) {
	out := bytes.NewBuffer(nil)
	for _, path := range paths {
		absDir, err := filepath.Abs(filepath.Dir(path))
		if err != nil {
			return nil, err
		}
		if _, err := os.Stat(absDir); os.IsNotExist(err) {
			return nil, err
		}
		files, err := os.ReadDir(absDir)
		if err != nil {
			return nil, err
		}
		for _, fi := range files {
			if fi.IsDir() {
				continue
			}
			absFile := filepath.Join(absDir, fi.Name())
			if matched, _ := filepath.Match(filepath.Join(absDir, filepath.Base(path)), absFile); matched {
				tmpContent, err := GetRenderedConfFromFile(absFile)
				if err != nil {
					return nil, fmt.Errorf("render extra config %s error: %v", absFile, err)
				}
				out.Write(tmpContent)
				out.WriteString("\n")
			}
		}
	}
	return out.Bytes(), nil
}
