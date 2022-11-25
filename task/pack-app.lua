-- enable/disable progress log
logrec = false
logdir = false

-- get frontend data directory
local rootdir = path.join(scrdir, "..", "frontend").."/"

-- check up deployment
if not checkfile(path.join(rootdir, "plugin")) then
	error"plugins does not installed, run 'task/deploy-plugins' script"
end
if not checkfile(path.join(rootdir, "build/app.bundle.js")) then
	error"frontend application bundle does not builded, run 'task/cc.base' script"
end
if not checkfile(path.join(rootdir, "build/main.bundle.js")) then
	error"frontend pages bundle does not builded, run 'task/cc.page' script"
end

-- write to log formatted string
local function logfmt(...)
	log(string.format(...))
end
-- format path with environment variables
local function envfmt(p)
	return path.toslash(string.gsub(p, "%${(%w+)}", function(varname)
		return os.getenv(varname)
	end))
end

-- inits new package
local pkg = wpk.new()
pkg.automime = true -- put MIME type for each file if it is not given explicit
pkg.secret = "hms-package" -- private key to sign cryptographic hashes for each file
pkg.crc32 = true -- generate CRC32 Castagnoli code for each file
pkg.sha256 = true -- generate SHA256 hash for each file
pkg:setinfo({ -- set package info
	label = "hms-app",
	link = "github.com/schwarzlichtbezirk/hms"
})

-- open wpk-file for write
pkg:begin(envfmt("${GOPATH}/bin/hms-app.wpk"))

-- write record log
local function logfile(kpath)
	logfmt("#%d %s, %d bytes, %s",
		pkg:gettag(kpath, "fid").uint32, kpath,
		pkg:filesize(kpath), pkg:gettag(kpath, "mime").string)
end
-- patterns for ignored files
local skippattern = {
	"^resmodel%.json$", -- skip original script
	"^thumb%.db$",
}
-- extensions of files that should not be included to package
local skipext = {
	wpk = true,
	sys = true,
	tmp = true,
	bak = true,
	-- compiler intermediate output
	log = true, tlog = true, lastbuildstate = true, unsuccessfulbuild = true,
	obj = true, lib = true, res = true,
	ilk = true, idb = true, ipdb = true, iobj = true, pdb = true, pgc = true, pgd = true,
	pch = true, ipch = true,
	cache = true,
}
-- check file names can be included to package
local function checkname(name)
	local fc = string.sub(name, 1, 1) -- first char
	if fc == "." or fc == "~" then return false end
	name = string.lower(name)
	for i, pattern in ipairs(skippattern) do
		if string.match(name, pattern) then return false end
	end
	local ext = string.match(name, "%.(%w+)$") -- file extension
	if ext and skipext[ext] then return false end
	return true
end
-- pack given directory and add to each file name given prefix
local function commonput(kpath, fpath)
	pkg:putfile(kpath, fpath)
	if logrec then logfile(kpath) end
end
local function authput(kpath, fpath)
	pkg:putfile(kpath, fpath)
	pkg:settag(kpath, "author", "schwarzlichtbezirk")
	if logrec then logfile(kpath) end
end
local function packdir(prefix, dir, putfunc)
	for i, name in ipairs(path.enum(dir)) do
		local kpath = path.join(prefix, name)
		local fpath = path.join(dir, name)
		local access, isdir = checkfile(fpath)
		if access and checkname(name) then
			if isdir then
				packdir(kpath, fpath, putfunc)
			else
				putfunc(kpath, fpath)
			end
		end
	end
	if logdir then logfmt("packed dir %s", dir) end
end

if logdir then logfmt("writes %s package", pkg.pkgpath) end

-- put some directories as is
packdir("build", rootdir.."build", commonput)
packdir("devmode", rootdir.."devmode", authput)
packdir("plugin", rootdir.."plugin", commonput)
packdir("tmpl", rootdir.."tmpl", commonput)
packdir("task", scrdir, commonput)
-- put sources
for i, fpath in ipairs{path.glob(rootdir.."../*")} do
	local has, isdir = checkfile(fpath)
	if not isdir then
		local fname = string.match(fpath, "([%w%-%.]+)$")
		authput("src/"..fname, fpath)
	end
end

if logdir then
	logfmt("packaged %d files to %d aliases on %d bytes", pkg.recnum, pkg.tagnum, pkg.datasize)
else
	logfmt("%s package: %d files, %d aliases, %d bytes", pkg.pkgpath, pkg.recnum, pkg.tagnum, pkg.datasize)
end

-- write records table, tags table and finalize wpk-file
pkg:finalize()
