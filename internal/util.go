package util

import (
	"github.com/zeromicro/go-zero/core/logx"

)

var verbose bool

func SetVerbose(v bool) {
	verbose = v
}

func PrintDebug(str string){
	if verbose{
		logx.Debug("\n______________________________\n")
		logx.Debug(str)
		logx.Debug("\n______________________________\n")
	}
	
}