/*
 * Copyright 2024 Gabriel Cataldo
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package log

import (
	"fmt"
	"github.com/GabrielHCataldo/go-logger/logger"
	"github.com/tech4works/gopen-gateway/internal/app"
	"os"
)

type bootLog struct {
	tag string
}

func NewBoot() app.BootLog {
	return bootLog{
		tag: "APP",
	}
}

func (l bootLog) PrintLogo() {
	fmt.Printf(` 
 ######    #######  ########  ######## ##    ##
##    ##  ##     ## ##     ## ##       ###   ##
##        ##     ## ##     ## ##       ####  ##
##   #### ##     ## ########  ######   ## ## ##
##    ##  ##     ## ##        ##       ##  ####
##    ##  ##     ## ##        ##       ##   ###
 ######    #######  ##        ######## ##    ##
-----------------------------------------------
Best open source API Gateway!            %s
-----------------------------------------------
2024 • Gabriel Cataldo.

`, os.Getenv("VERSION"))
}

func (l bootLog) PrintTitle(title string) {
	l.PrintInfof("-----------------------< %s%s%s >-----------------------", logger.StyleBold, title, logger.StyleReset)
}

func (l bootLog) PrintInfo(msg ...any) {
	Print(InfoLevel, l.tag, "", msg...)
}

func (l bootLog) PrintInfof(format string, msg ...any) {
	Printf(InfoLevel, l.tag, "", format, msg...)
}

func (l bootLog) PrintWarn(msg ...any) {
	Print(WarnLevel, l.tag, "", msg...)
}

func (l bootLog) PrintWarnf(format string, msg ...any) {
	Printf(WarnLevel, l.tag, "", format, msg...)
}

func (l bootLog) PrintError(msg ...any) {
	Print(ErrorLevel, l.tag, "", msg...)
}

func (l bootLog) SkipLine() {
	fmt.Println()
}
