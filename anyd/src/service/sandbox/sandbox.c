#include "_obj/_cgo_export.h"
#include "sandbox.h"
#include <stdlib.h>
#include <error.h>
#include <stdio.h>

extern int get_uri_path(lua_State *L);
extern int read_body_data(lua_State *L);
extern int write_data(lua_State *L);

void *init_lua(){
    lua_State *L = luaL_newstate();
    luaL_openlibs(L);
    lua_newtable(L);

    lua_pushcfunction(L, get_uri_path);
    // lua_setglobal(L, "get_uri_path");
    lua_setfield(L, -2, "get_uri_path");
    lua_pushcfunction(L, read_body_data);
    lua_setfield(L, -2, "read_body_data");
    // lua_setglobal(L, "read_body_data");
    lua_pushcfunction(L, write_data);
    lua_setfield(L, -2, "write");
    // lua_setglobal(L, "write");
    lua_setglobal(L, "go");
    return L;
}

int load_lua_file(void *p_luaCtx, const char *p_pszFilename){
    lua_State *L = (lua_State*)p_luaCtx;
    if( luaL_loadfile(L, p_pszFilename) || 
        lua_pcall(L, 0, 0, 0) ){
        error(0, (int)L, "load lua file failed, error: %s", lua_tostring(L, -1));
        return -1;
    }
    return 0;
}

int process_request(void *p_luaCtx, void *p_reqCtx){
    //push p_reqCtx to stack
    lua_State *L = (lua_State*)p_luaCtx;
    lua_getglobal(L, "process_request");
    lua_pushlightuserdata(L, p_reqCtx);
    if( lua_pcall(L, 1, 0, 0) ){
        error(0, (int)L, "call process_request failed, error:%s", lua_tostring(L, -1));
        return -1;
    }
    return 0;
}

void uninit(void *p_luaCtx){
    lua_State *L = (lua_State*)p_luaCtx;
    lua_close(L);
}

//interfaces for lua call.
int get_uri_path(lua_State *L){
    //get reqCtx as the stack parameter!
    int param_count = lua_gettop(L);
    if ( param_count != 1 ){
        return luaL_error(L, "invalid parameter count!");
    }
    void *reqCtx = lua_touserdata(L, -1);
    lua_pop(L, 1);
    char *uri_path = GetURIPath(reqCtx);
    lua_pushstring(L, uri_path);
    free(uri_path);
    return 1;
}

int read_body_data(lua_State *L){
    int param_count = lua_gettop(L);
    if ( param_count != 1 ){
        return luaL_error(L, "invalid parameter count!");
    }
    void *reqCtx = lua_touserdata(L, -1);
    lua_pop(L, 1);
    struct ReadBodyData_return ret = ReadBodyData(reqCtx);
    lua_pushlstring(L, ret.r0, ret.r1);
    free(ret.r0);
    return 1;
}

int write_data(lua_State *L){
    int param_count = lua_gettop(L);
    if ( param_count != 2 ){
        return luaL_error(L, "invalid parameter count!");
    }
    // void *reqCtx = luaL_checkudata(L, 1, NULL);

    void *reqCtx = lua_touserdata(L, 1);
    if ( reqCtx == NULL ){
        printf("request context is a null pointer!\n");
        return -1;
    }
    // lua_pop(L, 1);
    size_t len = 0;
    const char *data = lua_tolstring(L, 2, &len);
    lua_pop(L, 2);
    // const char *data = luaL_checklstring(L, 2, &len);
    int written = WriteData(reqCtx, (char *)data, len);
    lua_pushnumber(L, written);
    return 1;
}
