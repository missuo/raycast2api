/*
 * @Author: Vincent Yang
 * @Date: 2025-04-04 16:14:09
 * @LastEditors: Vincent Yang
 * @LastEditTime: 2025-04-09 16:17:12
 * @FilePath: /raycast2api/main.go
 * @Telegram: https://t.me/missuo
 * @GitHub: https://github.com/missuo
 *
 * Copyright © 2025 by Vincent, All Rights Reserved.
 */

package main

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/missuo/raycast2api/service"
)

// Main function
func main() {
	config := service.InitConfig()

	fmt.Printf("Raycast2API has been successfully launched! Listening on %v\n", config.Port)

	// Set Release Mode
	gin.SetMode(gin.ReleaseMode)

	app := service.Router(config)
	app.Run(fmt.Sprintf(":%v", config.Port))
}
