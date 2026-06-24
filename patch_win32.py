import re

with open("runtime/wolf_runtime.c", "r") as f:
    content = f.read()

helpers = """
#ifdef _WIN32
static WCHAR* wolf_utf8_to_utf16(const char* utf8) {
    if (!utf8) return NULL;
    int len = MultiByteToWideChar(CP_UTF8, 0, utf8, -1, NULL, 0);
    if (len == 0) return NULL;
    WCHAR* utf16 = (WCHAR*)wolf_req_alloc(len * sizeof(WCHAR));
    MultiByteToWideChar(CP_UTF8, 0, utf8, -1, utf16, len);
    return utf16;
}

static char* wolf_utf16_to_utf8(const WCHAR* utf16) {
    if (!utf16) return NULL;
    int len = WideCharToMultiByte(CP_UTF8, 0, utf16, -1, NULL, 0, NULL, NULL);
    if (len == 0) return NULL;
    char* utf8 = (char*)wolf_req_alloc(len);
    WideCharToMultiByte(CP_UTF8, 0, utf16, -1, utf8, len, NULL, NULL);
    return utf8;
}
#endif
"""

# Insert helpers right after #include <windows.h> if it's there
if "static WCHAR* wolf_utf8_to_utf16" not in content:
    content = re.sub(r'(#include <windows\.h>\n)', r'\1' + helpers, content)

# 1. wolf_scan_dir
scan_dir_old = """    WIN32_FIND_DATAA findFileData;
    char searchPath[4096];
    snprintf(searchPath, sizeof(searchPath), "%s\\\\*", path);
    HANDLE hFind = FindFirstFileA(searchPath, &findFileData);
    if (hFind != INVALID_HANDLE_VALUE) {
        do {
            if (strcmp(findFileData.cFileName, ".") == 0 || strcmp(findFileData.cFileName, "..") == 0)
                continue;
            if (!first) buf[pos++] = ',';
            first = 0;
            buf[pos++] = '"';
            const char* name = findFileData.cFileName;"""

scan_dir_new = """    WIN32_FIND_DATAW findFileData;
    char searchPath[4096];
    snprintf(searchPath, sizeof(searchPath), "%s\\\\*", path);
    WCHAR* wSearchPath = wolf_utf8_to_utf16(searchPath);
    HANDLE hFind = FindFirstFileW(wSearchPath, &findFileData);
    if (hFind != INVALID_HANDLE_VALUE) {
        do {
            if (wcscmp(findFileData.cFileName, L".") == 0 || wcscmp(findFileData.cFileName, L"..") == 0)
                continue;
            if (!first) buf[pos++] = ',';
            first = 0;
            buf[pos++] = '"';
            char* utf8Name = wolf_utf16_to_utf8(findFileData.cFileName);
            const char* name = utf8Name;"""

content = content.replace(scan_dir_old, scan_dir_new)
content = content.replace("FindNextFileA", "FindNextFileW")

# 2. wolf_file_list_dir
list_dir_old = """    WIN32_FIND_DATAA findFileData;
    char searchPath[4096];
    snprintf(searchPath, sizeof(searchPath), "%s\\\\*", path);
    HANDLE hFind = FindFirstFileA(searchPath, &findFileData);
    if (hFind != INVALID_HANDLE_VALUE) {
        do {
            if (strcmp(findFileData.cFileName, ".") == 0 || strcmp(findFileData.cFileName, "..") == 0)
                continue;
            wolf_array_push(arr, wolf_req_strdup(findFileData.cFileName));
        } while (FindNextFileA(hFind, &findFileData) != 0);"""

list_dir_new = """    WIN32_FIND_DATAW findFileData;
    char searchPath[4096];
    snprintf(searchPath, sizeof(searchPath), "%s\\\\*", path);
    WCHAR* wSearchPath = wolf_utf8_to_utf16(searchPath);
    HANDLE hFind = FindFirstFileW(wSearchPath, &findFileData);
    if (hFind != INVALID_HANDLE_VALUE) {
        do {
            if (wcscmp(findFileData.cFileName, L".") == 0 || wcscmp(findFileData.cFileName, L"..") == 0)
                continue;
            char* utf8Name = wolf_utf16_to_utf8(findFileData.cFileName);
            wolf_array_push(arr, wolf_req_strdup(utf8Name));
        } while (FindNextFileW(hFind, &findFileData) != 0);"""

content = content.replace(list_dir_old, list_dir_new)

# 3. _stat wrappers
stat_old = """struct _stat st;
    return (_stat(path, &st) == 0 && (st.st_mode & _S_IFDIR));"""
stat_new = """struct _stat st;
    WCHAR* wPath = wolf_utf8_to_utf16(path);
    return (_wstat(wPath, &st) == 0 && (st.st_mode & _S_IFDIR));"""
content = content.replace(stat_old, stat_new)

file_stat_old = """struct _stat st;
    return (_stat(path, &st) == 0 && (st.st_mode & _S_IFREG));"""
file_stat_new = """struct _stat st;
    WCHAR* wPath = wolf_utf8_to_utf16(path);
    return (_wstat(wPath, &st) == 0 && (st.st_mode & _S_IFREG));"""
content = content.replace(file_stat_old, file_stat_new)

file_size_old = """struct _stat st;
    if (_stat(path, &st) == 0) {
        return (double)st.st_size;"""
file_size_new = """struct _stat st;
    WCHAR* wPath = wolf_utf8_to_utf16(path);
    if (_wstat(wPath, &st) == 0) {
        return (double)st.st_size;"""
content = content.replace(file_size_old, file_size_new)

# 4. _mkdir
mkdir_old = "if (_mkdir(tmp) != 0 && errno != EEXIST) return 0;"
mkdir_new = """WCHAR* wTmp = wolf_utf8_to_utf16(tmp);
            if (_wmkdir(wTmp) != 0 && errno != EEXIST) return 0;"""
content = content.replace(mkdir_old, mkdir_new)

mkdir2_old = "return (_mkdir(tmp) == 0 || errno == EEXIST) ? 1 : 0;"
mkdir2_new = """WCHAR* wTmp2 = wolf_utf8_to_utf16(tmp);
    return (_wmkdir(wTmp2) == 0 || errno == EEXIST) ? 1 : 0;"""
content = content.replace(mkdir2_old, mkdir2_new)


with open("runtime/wolf_runtime.c", "w") as f:
    f.write(content)
print("win32 pal fixed")
