import ctypes as c
import ctypes.wintypes as win
import ntpath
import orca.enums as e
import os, sys
import datetime
import re
from orca.source import Src
from pathlib import Path

class SessionConfig(c.Structure):
	_fields_ = [
		('eClobber', c.c_uint), #1=PBORCA_CLOBBER
		('eExportEncoding', c.c_uint), #0=UTF16LE, 1=UTF8
		('bexport_header', win.BOOL),
		('bExportIncludeBinary', win.BOOL),
		('bExportCreateFile', win.BOOL),
		('pExportDirectory', win.LPWSTR),
		('eImportEncoding', c.c_uint),
		('bDebug', win.BOOL)
	]

	def __init__(self):
		self.eClobber = 1
		self.eExportEncoding = e.PBEncoding.UNICODE.value
		self.bexport_header = True
		self.bExportIncludeBinary = True
		self.bExportCreateFile = False
		self.pExportDirectory = (win.LPWSTR)("")
		self.eImportEncoding = e.PBEncoding.UNICODE.value
		self.bDebug = False

class Orca:
	pSessionOpen = c.WINFUNCTYPE(c.c_void_p)
	pSessionClose = c.WINFUNCTYPE(c.c_void_p, c.c_void_p)
	aSessionClose = (e.PBArg.INPUT.value, "session_handle"),
	def __init__(self, dll_path : Path = Path("C:\\Program Files (x86)\\Appeon\\Shared\\PowerBuilder\\PBORC170.DLL")):
		self.dll = c.WinDLL(os.fspath(dll_path))
		
		self.fSessionOpen = Orca.pSessionOpen(("PBORCA_SessionOpen", self.dll))
		self.fSessionClose = Orca.pSessionClose(("PBORCA_SessionClose", self.dll), self.aSessionClose)
		
		self.session_handle = self.fSessionOpen()
		self.config  = SessionConfig()
		self.configure()
		
	def __del__(self):
		self.fSessionClose(c.c_void_p(self.session_handle))
	
	def configure(self, encoding : e.PBEncoding = None, export_header : bool = None,
					export_binary_data : bool = None, export_folder : str = None):
		
		if self.config == None:
			self.config = SessionConfig()

		if encoding != None:
			self.config.eExportEncoding = encoding.value
			self.config.eImportEncoding = encoding.value
		if export_header != None:
			self.config.bexport_header = export_header
		if export_binary_data != None:
			self.config.bExportIncludeBinary = export_binary_data
		if export_folder != None:
			self.config.bExportCreateFile = True
			self.config.pExportDirectory = export_folder
		return e.PBResult(self.dll.PBORCA_ConfigureSession(
			self.session_handle,
			c.byref(self.config)
		))

	def read_source(self, source_path : Path):
		with open(source_path, 'r', encoding='utf-8-sig') as f:
			return f.read()

	def pbl_create(self, pbl_path : Path, pbl_comment : str):
		return e.PBResult(
			self.dll.PBORCA_LibraryCreate(self.session_handle, os.fspath(pbl_path), pbl_comment)
		)

	def set_pbl_list(self, pbl_path_list : list[Path]):
		#pbl_list = pbl_list[::-1] #TODO: Why ::-1 ?
		pbl_array = (win.LPWSTR * len(pbl_path_list))(*list(map(lambda o : os.fspath(o), pbl_path_list)))
		
		return e.PBResult(
			self.dll.PBORCA_SessionSetLibraryList(self.session_handle, pbl_array, len(pbl_path_list))
		)

	def set_current_app(self, app_pbl_path : Path, app_name : str):
		return e.PBResult(
			self.dll.PBORCA_SessionSetCurrentAppl(self.session_handle, os.fspath(app_pbl_path), app_name)
		)
	def import_sources(self, source_files : list[Src], error_list : list[str] = []):
		self.return_list = error_list
		pbl_pathstrings = []
		entry_names = []
		entry_types = []
		entry_comments = []
		entry_sources = []
		entry_source_lengths = []
		number_of_entries = 0
		for source_file in source_files:
			pbl_pathstrings.append(os.fspath(source_file.pbl_path))
			entry_names.append(source_file.name)
			entry_types.append(source_file.src_type.value)
			entry_source = self.read_source(source_file.own_path)
			
			
			entry_sources.append((win.LPWSTR)(entry_source))
			entry_source_lengths.append(len(entry_source)*2)
			entry_comments.append(self._get_comment_from_source(entry_source))
			number_of_entries += 1
		#TODO: Import also binary part
		return e.PBResult(self.dll.PBORCA_CompileEntryImportList(
			self.session_handle,
			(win.LPWSTR * number_of_entries)(*pbl_pathstrings), #pLibraryNames
			(win.LPWSTR * number_of_entries)(*entry_names), #pEntryNames
			(c.c_long * number_of_entries)(*entry_types), #otEntryTypes
			(win.LPWSTR * number_of_entries)(*entry_comments), #pComments
			(win.LPWSTR * number_of_entries)(*entry_sources), #pEntrySyntaxBuffers,
			(c.c_long * number_of_entries)(*entry_source_lengths), #pEntrySyntaxBuffSizes,
			number_of_entries,
			fOrcaCompErr(self._callback_compilation_error),
			0
		))
				
	@classmethod
	def _get_comment_from_source(cls, sSource):
		return re.search(r'(?<=(\r\n\$PBExportComments\$)).*?(?=(\r\n))|$', sSource[:500]).group(0)

	def import_source(self, pbl_path : Path, source_path : Path, error_list : list[str] = []):
		self.return_list = error_list
		entry_source = self.read_source(source_path).decode('utf-8-sig')
		
		return e.PBResult(self.dll.PBORCA_CompileEntryImport(
			self.session_handle,
			(win.LPWSTR)(os.fspath(pbl_path)), #lpszLibraryName
			(win.LPWSTR)(source_path.stem), #lpszEntryName
			(c.c_long)(e.PBSrcType.get_type(source_path).value), #otEntryType
			(win.LPWSTR)(self._get_comment_from_source(entry_source)), #lpszComments
			(win.LPWSTR)(entry_source), #entry_source, #lpszEntrySyntax
			(c.c_long)(len(entry_source.encode('utf-16-le'))), #lEntrySyntaxBuffSize
			fOrcaCompErr(self._callback_compilation_error),
			0
		))	
	
	def get_source_list(self, pbl_path : Path) -> (e.PBResult, list[list[str]]):
		self.return_list = []
		lpszLibName = (win.LPWSTR)(os.fspath(pbl_path))
		lpszLibComments = (win.LPWSTR)("".ljust(e.PBORCA_MSGBUFFER + 1))
		iCmntsBuffLen = (c.c_int)(e.PBORCA_MSGBUFFER + 1)
		ret = e.PBResult(self.dll.PBORCA_LibraryDirectory(
			self.session_handle,
			lpszLibName,
			lpszLibComments,
			iCmntsBuffLen,
			fOrcaDirEntry(self._callback_dir_entry),
			0
		))
		return (ret, self.return_list)

	def export_source(self, pbl_path : Path, entry_name : str, entry_type : e.PBSrcType, entry_size) -> (e.PBResult, Src):
		src_source = (win.LPWSTR)("".ljust(entry_size)) #lpszExportBuffer
		ret = e.PBResult(self.dll.PBORCA_LibraryEntryExport(
			self.session_handle,
			(win.LPWSTR)(os.fspath(pbl_path)), #lpszLibraryName 
			(win.LPWSTR)(entry_name), #lpszEntryName
			entry_type.value,
			src_source,
			entry_size
		))
		
		return (ret, Src(source = src_source.value.rstrip(), src_type = entry_type, name = entry_name, pbl_path = pbl_path))
	
	def get_entry_info(self, pbl_path : Path, entry, entry_type : e.PBSrcType) -> (e.PBResult, list[str]):
		pEntryInformationBlock = PBORCA_EntryInfo()
		ret = e.PBResult(self.dll.PBORCA_LibraryEntryInformation(
			self.session_handle,
			(win.LPWSTR)(os.fspath(pbl_path)),
			(win.LPWSTR)(entry),
			entry_type.value,
			c.byref(pEntryInformationBlock)
		))
		return (ret, [
			pEntryInformationBlock.szComments,
			pEntryInformationBlock.lCreateTime,
			pEntryInformationBlock.lObjectSize,
			pEntryInformationBlock.lSourceSize,
		])

	# Callback function for compilation errors while source imports
	def _callback_compilation_error(self, a, b):
		#print("orcaCompErr", a.contents.errorLevel, a.contents.msgNr, a.contents.msgText) 
		self.return_list.append([
			a.contents.errorLevel,
			a.contents.msgNr,
			a.contents.msgText,
			a.contents.colNr,
			a.contents.lineNr
		])
	
	# Callback function for source list result
	def _callback_dir_entry(self, a, b):
		self.return_list.append([
			a.contents.lpszEntryName, 
			e.PBSrcType(a.contents.otEntryType),
			a.contents.entry_size,
			datetime.datetime.fromtimestamp(a.contents.lCreateTime).strftime('%Y-%m-%d %H:%M:%S'),
			a.contents.szComments
		])

# Callback function prototype for source import
class PBORCA_COMPERR(c.Structure):
	_fields_ = [
		('errorLevel', c.c_int),
		('msgNr', win.LPWSTR),
		('msgText', win.LPWSTR),
		('colNr', c.c_uint),
		('lineNr', c.c_uint)
	]
fOrcaCompErr = c.WINFUNCTYPE(None, c.POINTER(PBORCA_COMPERR), c.c_void_p)

# Callback function prototype for source list
class PBORCA_DIRENTRY(c.Structure):
	_fields_ = [
		('szComments', (win.WCHAR * (e.PBORCA_MAXCOMMENT + 1))),
		('lCreateTime', c.c_long),
		('entry_size', c.c_long),
		('lpszEntryName', win.LPWSTR),
		('otEntryType', c.c_int)
	]
fOrcaDirEntry = c.WINFUNCTYPE(None, c.POINTER(PBORCA_DIRENTRY), c.c_void_p)

# Struct for describing a powerbuilder class
class PBORCA_EntryInfo(c.Structure):
	_fields_ = [
		('szComments', (win.WCHAR * (e.PBORCA_MAXCOMMENT + 1))),
		('lCreateTime', c.c_long),
		('lObjectSize', c.c_long),
		('lSourceSize', c.c_long)
	]
			
