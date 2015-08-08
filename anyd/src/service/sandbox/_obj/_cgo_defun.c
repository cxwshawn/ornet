
#include "runtime.h"
#include "cgocall.h"

void ·_Cerrno(void*, int32);

void
·_Cfunc_GoString(int8 *p, String s)
{
	s = runtime·gostring((byte*)p);
	FLUSH(&s);
}

void
·_Cfunc_GoStringN(int8 *p, int32 l, String s)
{
	s = runtime·gostringn((byte*)p, l);
	FLUSH(&s);
}

void
·_Cfunc_GoBytes(int8 *p, int32 l, Slice s)
{
	s = runtime·gobytes((byte*)p, l);
	FLUSH(&s);
}

void
·_Cfunc_CString(String s, int8 *p)
{
	p = runtime·cmalloc(s.len+1);
	runtime·memmove((byte*)p, s.str, s.len);
	p[s.len] = 0;
	FLUSH(&p);
}

void
·_Cfunc__CMalloc(uintptr n, int8 *p)
{
	p = runtime·cmalloc(n);
	FLUSH(&p);
}

#pragma cgo_export_dynamic GetURIPath
extern void ·GetURIPath();

#pragma cgo_export_static _cgoexp_30ddf24dc61f_GetURIPath
#pragma textflag 7
void
_cgoexp_30ddf24dc61f_GetURIPath(void *a, int32 n)
{
	runtime·cgocallback(·GetURIPath, a, n);
}
#pragma cgo_export_dynamic ReadBodyData
extern void ·ReadBodyData();

#pragma cgo_export_static _cgoexp_30ddf24dc61f_ReadBodyData
#pragma textflag 7
void
_cgoexp_30ddf24dc61f_ReadBodyData(void *a, int32 n)
{
	runtime·cgocallback(·ReadBodyData, a, n);
}
#pragma cgo_export_dynamic WriteData
extern void ·WriteData();

#pragma cgo_export_static _cgoexp_30ddf24dc61f_WriteData
#pragma textflag 7
void
_cgoexp_30ddf24dc61f_WriteData(void *a, int32 n)
{
	runtime·cgocallback(·WriteData, a, n);
}
