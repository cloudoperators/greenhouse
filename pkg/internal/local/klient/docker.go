// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Greenhouse contributors
// SPDX-License-Identifier: Apache-2.0

package klient

import (
	"github.com/cloudoperators/greenhouse/pkg/internal/local/utils"
)

// BuildImage - uses docker cli to build an image
func BuildImage(img, platform, dockerFilePath string) error {
	return utils.Shell{
		Cmd: "docker build --platform ${platform} -t ${img} ${path}",
		Vars: map[string]string{
			"path":     dockerFilePath,
			"img":      img,
			"platform": platform,
		},
	}.Exec()
}
