package main

import (
	"fmt"
	"github.com/360EntSecGroup-Skylar/excelize"
	"unicode/utf8"
)

type excel struct {
	*excelize.File
}

func trimSheetName(name string) string {
	var r []rune
	for _, v := range name {
		switch v {
		case 58, 92, 47, 63, 42, 91, 93: // replace :\/?*[]
			continue
		default:
			r = append(r, v)
		}
	}
	name = string(r)
	if utf8.RuneCountInString(name) > 31 {
		name = string([]rune(name)[0:31])
	}
	return name
}

func (self *excel) register(openId, student, gender, phone, category, session string) {
	sheetName := trimSheetName(session)
	index := self.NewSheet(sheetName)
	self.InsertRow(sheetName, 1)
	self.SetCellValue(sheetName, "A1", openId)
	self.SetCellValue(sheetName, "B1", student)
	self.SetCellValue(sheetName, "C1", gender)
	self.SetCellValue(sheetName, "D1", phone)
	self.SetCellValue(sheetName, "E1", category)
	if self.GetColWidth(sheetName, "A") != 20 {
		self.SetColWidth(sheetName, "A", "E", 20)
	}
	self.SetActiveSheet(index)
	self.DeleteSheet("Sheet1")
	self.Save()
}

func InitReport(eventName string) (*excel, error) {
	filename := fmt.Sprintf(systembasePath+"/report/%s.xlsx", eventName)
	xlsx, err := excelize.OpenFile(filename)
	if err != nil {
		xlsx = excelize.NewFile()
		xlsx.SaveAs(filename)
		xlsx, err = excelize.OpenFile(filename)
		if err != nil {
			return nil, err
		}
	}

	return &excel{xlsx}, nil
}
