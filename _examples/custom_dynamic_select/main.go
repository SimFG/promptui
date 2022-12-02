package main

import (
	"fmt"
	"github.com/manifoldco/promptui/list"
	"strings"

	"github.com/manifoldco/promptui"
)

type Ui struct {
	Name              string
	Desc              string
	CmdConfig         *CmdConfig
	CmdParam          *CmdParam
	isAnyParamValueUi bool
}

var ExitUi = Ui{
	Name: "🔥 Execute",
	Desc: "End the command",
}

func GetValueUi(value string) Ui {
	return Ui{
		Name: value,
		Desc: "Param value",
	}
}

type CmdParamState int

const (
	None CmdParamState = iota
	Name
	Value
)

type GetCmdParamValues func(i string) []Ui

var AnyParamValueUi = Ui{
	Name:              "? Any Param Value",
	Desc:              "Enter any value",
	isAnyParamValueUi: true,
}

// AnyParamValues TODO fubang need check
var AnyParamValues = func(i string) []Ui {
	return []Ui{AnyParamValueUi}
}

type CmdParam struct {
	Name          string
	Value         string
	Desc          string
	Complete      GetCmdParamValues
	State         CmdParamState
	isNoNameParam bool
}

// GetNoNameCmdParams TODO fubang done
func GetNoNameCmdParams(count int) []*CmdParam {
	res := make([]*CmdParam, 0, count)
	for i := 0; i < count; i++ {
		res = append(res, &CmdParam{
			Name:          "? No Name Param",
			Desc:          "Enter any value",
			Complete:      AnyParamValues,
			State:         Name,
			isNoNameParam: true,
		})
	}
	return res
}

type CmdConfig struct {
	Name     string
	Desc     string
	Params   []*CmdParam
	Children []*CmdConfig
}

func main() {
	templates := &promptui.SelectTemplates{
		Label:    "{{ . }}?",
		Active:   "\U0001F336 {{ .Name | cyan }}",
		Inactive: "  {{ .Name | cyan }}",
		Selected: "\U0001F336 {{ .Name | red | cyan }}",
		Details:  `{{ .Name | bold }}: {{ .Desc }}`,
	}

	cmd := &CmdConfig{
		Name: "etcdctl",
		Desc: "etcd control interface",
		Params: []*CmdParam{
			{Name: "endpoint", Desc: "etcd address", Complete: AnyParamValues},
		},
		Children: []*CmdConfig{
			{Name: "get", Desc: "get the value", Params: GetNoNameCmdParams(1)},
			{Name: "get2", Desc: "get the value", Params: GetNoNameCmdParams(2)},
			{Name: "put", Desc: "put the value", Params: []*CmdParam{
				{Name: "value", Desc: "the data string", Complete: func(i string) []Ui {
					// TODO more values
					return []Ui{
						GetValueUi("put1"),
						GetValueUi("put2"),
					}
				}},
			}},
		},
	}
	currentCmd := cmd
	backspaceCount := 0
	anyInput := ""

	var uis []Ui
	uis = append(uis, ExitUi)
	for _, param := range cmd.Params {
		uis = append(uis, Ui{Name: "--" + param.Name, Desc: param.Desc, CmdParam: param})
	}
	for _, child := range cmd.Children {
		uis = append(uis, Ui{Name: child.Name, Desc: child.Desc, CmdConfig: child})
	}

	searcher := func(input string, index int) bool {
		lastSpace := strings.LastIndex(input, " ")
		if lastSpace != -1 {
			input = input[lastSpace+1:]
		}
		ui := uis[index]
		if ui.isAnyParamValueUi {
			anyInput = input
			return true
		}
		name := strings.Replace(strings.ToLower(ui.Name), " ", "", -1)
		input = strings.Replace(strings.ToLower(input), " ", "", -1)

		backspaceCount = len(input)
		return strings.Contains(name, input)
	}

	current := func() (CmdParamState, *CmdParam) {
		currentState := None
		var currentParam *CmdParam
		for _, param := range currentCmd.Params {
			if param.State == Name {
				currentState = param.State
				currentParam = param
				break
			}
		}
		return currentState, currentParam
	}

	line := ""

	// TODO async loading
	itemGenerator := func(input string) []*interface{} {
		// TODO next command if the last char of the input is the space

		uis = uis[:0]
		currentState, currentParam := current()

		switch currentState {
		case None:
			uis = append(uis, ExitUi)
			for _, param := range currentCmd.Params {
				if param.State == None {
					uis = append(uis, Ui{Name: "--" + param.Name, Desc: param.Desc, CmdParam: param})
				}
			}
			for _, child := range currentCmd.Children {
				uis = append(uis, Ui{Name: child.Name, Desc: child.Desc, CmdConfig: child})
			}
		case Name:
			tmp := strings.Split(strings.TrimSpace(input), " ")
			uis = currentParam.Complete(tmp[len(tmp)-1])
			for i, _ := range uis {
				uis[i].CmdParam = currentParam
			}
		}

		return list.GetItemFromInterface(uis)
	}

	exitFunc := func(i, j int) (data []byte, exit bool) {
		selectUi := uis[j]
		exit = selectUi == ExitUi
		if !exit {
			if selectUi.CmdConfig != nil {
				currentCmd = selectUi.CmdConfig
				data = []byte(currentCmd.Name + " ")
			} else if selectUi.CmdParam != nil {
				selectUi.CmdParam.State += 1
				if selectUi.CmdParam.State == Name {
					data = []byte("--" + uis[j].CmdParam.Name + " ")
				} else if selectUi.isAnyParamValueUi {
					line += anyInput
					backspaceCount = 0
					data = []byte{' '}
				} else {
					data = []byte(selectUi.Name + " ")
				}
			}
			line += string(data)
			data = append(promptui.GetBackspace(backspaceCount), data...)
		}
		return
	}

	// TODO handle backspace
	prompt := promptui.Select{
		Items:                  uis,
		Templates:              templates,
		Size:                   4,
		Searcher:               searcher,
		ItemGenerator:          itemGenerator,
		ExitFunc:               exitFunc,
		SearchPrompt:           "etcdctl ",
		StartInSearchMode:      true,
		HideSelected:           true,
		HideLabel:              true,
		DisableExistSearchMode: true,
	}

	_, _, err := prompt.Run()

	if err != nil {
		fmt.Printf("Prompt failed %v\n", err)
		return
	}

	fmt.Printf("Line: %s\n", line)
}
