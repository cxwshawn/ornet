
function process_request(reqCtx)
    local uri_path = go.get_uri_path(reqCtx)
    print(uri_path)
    local count = go.write(reqCtx, uri_path)
    if count ~= string.len(uri_path) then
        print("write count is", count, "length of uri_path is", string.len(uri_path))
    end
end