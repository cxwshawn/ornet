/* Created by cgo - DO NOT EDIT. */
#include "_cgo_export.h"

extern void crosscall2(void (*fn)(void *, int), void *, int);

extern void _cgoexp_30ddf24dc61f_GetURIPath(void *, int);

char* GetURIPath(void* p0)
{
	struct {
		void* p0;
		char* r0;
	} __attribute__((__packed__, __gcc_struct__)) a;
	a.p0 = p0;
	crosscall2(_cgoexp_30ddf24dc61f_GetURIPath, &a, 16);
	return a.r0;
}
extern void _cgoexp_30ddf24dc61f_ReadBodyData(void *, int);

struct ReadBodyData_return ReadBodyData(void* p0)
{
	struct {
		void* p0;
		char* r0;
		GoInt r1;
	} __attribute__((__packed__, __gcc_struct__)) a;
	struct ReadBodyData_return r;
	a.p0 = p0;
	crosscall2(_cgoexp_30ddf24dc61f_ReadBodyData, &a, 24);
	r.r0 = a.r0;
	r.r1 = a.r1;
	return r;
}
extern void _cgoexp_30ddf24dc61f_WriteData(void *, int);

int WriteData(void* p0, char* p1, GoInt p2)
{
	struct {
		void* p0;
		char* p1;
		GoInt p2;
		int r0;
		char __pad0[4];
	} __attribute__((__packed__, __gcc_struct__)) a;
	a.p0 = p0;
	a.p1 = p1;
	a.p2 = p2;
	crosscall2(_cgoexp_30ddf24dc61f_WriteData, &a, 32);
	return a.r0;
}
