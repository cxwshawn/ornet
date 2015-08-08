#ifndef _SANDBOX_H_
#define _SANDBOX_H_

#include <lua.h>
#include <lauxlib.h>
#include <lualib.h>

// extern "C"{

//interfaces for golang call.
void *init_lua();
int load_lua_file(void *p_luaCtx, const char *p_pszFilename);
int process_request(void *p_luaCtx, void *p_reqCtx);
void uninit(void *p_luaCtx);
int get_uri_path(lua_State *L);
int read_body_data(lua_State *L);
int write_data(lua_State *L);

// }

#endif