// Created by cgo - DO NOT EDIT

package sandbox

import "unsafe"

import "syscall"

import _ "runtime/cgo"

type _ unsafe.Pointer

func _Cerrno(dst *error, x int32) { *dst = syscall.Errno(x) }

type _Ctype_char int8

type _Ctype_int int32

type _Ctype_void [0]byte

func _Cfunc_CString(string) *_Ctype_char
func _Cfunc_GoStringN(*_Ctype_char, _Ctype_int) string
