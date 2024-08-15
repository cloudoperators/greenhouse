// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package setup

import (
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
	"github.com/vladimirvivien/gexe"
)

func BuildImage(img, platform, dockerFilePath string) error {
	exec := gexe.New()
	exec.SetVar("path", dockerFilePath)
	exec.SetVar("img", img)
	exec.SetVar("platform", platform)
	utils.Logf("building %s image for platform %s", img, platform)
	cmd := exec.RunProc("docker build --platform ${platform} -t ${img} ${path}")
	if err := cmd.Err(); err != nil {
		utils.LogErr("error building docker image: %s", cmd.Result())
		return err
	}
	utils.Logf("%s \n docker image %s built successfully", cmd.Result(), img)
	return nil
}
