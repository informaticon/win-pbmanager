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
		('bExportHeader', win.BOOL),
		('bExportIncludeBinary', win.BOOL),
		('bExportCreateFile', win.BOOL),
		('pExportDirectory', win.LPWSTR),
		('eImportEncoding', c.c_uint),
		('bDebug', win.BOOL)
	]

	def __init__(self):
		self.eClobber = 1
		self.eExportEncoding = e.PBEncoding.UNICODE.value
		self.bExportHeader = True
		self.bExportIncludeBinary = True
		self.bExportCreateFile = False
		self.pExportDirectory = (win.LPWSTR)("")
		self.eImportEncoding = e.PBEncoding.UNICODE.value
		self.bDebug = False

class Orca:
	pSessionOpen = c.WINFUNCTYPE(c.c_void_p)
	pSessionClose = c.WINFUNCTYPE(c.c_void_p, c.c_void_p)
	aSessionClose = (e.PBArg.INPUT.value, "hSession"),
	def __init__(self, workDir : Path, dllPath : Path = Path("C:\\Program Files (x86)\\Appeon\\Shared\\PowerBuilder\\PBORC170.DLL")):
		self.dll = c.WinDLL(os.fspath(dllPath))
		#self.dll.set_callback(orcaCallback)
		#print(c.c_int.in_dll(self.dll, "PBORCA_P_CODE"))
		
		self.workDir = workDir

		self.fSessionOpen = Orca.pSessionOpen(("PBORCA_SessionOpen", self.dll))
		self.fSessionClose = Orca.pSessionClose(("PBORCA_SessionClose", self.dll), self.aSessionClose)
		#self.fLibraryCreate = self.pLibraryCreate(("PBORCA_LibraryCreate", self.dll), self.aLibraryCreate)

		self.hSession = self.fSessionOpen()
		self.config  = SessionConfig()
		self.configure()
		
	def __del__(self):
		self.fSessionClose(c.c_void_p(self.hSession))
	
	def configure(self, encoding : e.PBEncoding = None, exportHeader : bool = None,
					exportBinData : bool = None, exportFolder : str = None):
		
		if self.config == None:
			self.config = SessionConfig()

		if encoding != None:
			self.config.eExportEncoding = c.c_uint(encoding.value)
			self.config.eImportEncoding = c.c_uint(encoding.value)
		if exportHeader != None:
			self.config.bExportHeader = c.c_bool(exportHeader)
		if exportBinData != None:
			self.config.bExportIncludeBinary = c.c_bool(exportBinData)
		if exportFolder != None:
			self.config.bExportCreateFile = c.c_bool(True)
			self.config.pExportDirectory = (win.LPWSTR)(exportFolder)
		return e.PBResult(self.dll.PBORCA_ConfigureSession(
			self.hSession,
			c.byref(self.config)
		))

	def readSource(self, sFileFullPath):
		with open(sFileFullPath, 'rb') as f:
			return f.read()

	def libraryCreate(self, sLibraryName, sLibraryComment):
		sLibraryFullPath = self.workDir.joinpath(sLibraryName)
		'''
		p1 = c.c_void_p(self.hSession)
		p2 = win.LPSTR(sLibraryFullPath.encode(self.encoding))
		p3 = win.LPSTR(sLibraryComment.encode(self.encoding))
		return self.fLibraryCreate(p1, p2, p3)
		'''
		return e.PBResult(
			self.dll.PBORCA_LibraryCreate(self.hSession, os.fspath(sLibraryFullPath), sLibraryComment)
		)

	def setLibraryList(self, sLibList):
		sLibList = sLibList[::-1] 
		sLibList = list(map(lambda o : os.fspath(self.workDir.joinpath(o)), sLibList))
		sLibArr = (win.LPWSTR * len(sLibList))(*sLibList)
		
		return e.PBResult(
			self.dll.PBORCA_SessionSetLibraryList(self.hSession, sLibArr, len(sLibList))
		)

	def setApplication(self, sAppLibName, sAppName)	:
		return e.PBResult(
			self.dll.PBORCA_SessionSetCurrentAppl(self.hSession, os.fspath(self.workDir.joinpath(sAppLibName)), sAppName)
		)

	def sourceImportBatch(self, sourceFiles : Src, errorList = []):
		self.returnList = errorList
		sLibraries = []
		sEntryNames = []
		pEntryTypes = []
		sEntryComments = []
		sEntrySources = []
		lEntrySourceLengths = []
		iNumberOfEntries = 0
		for sourceFile in sourceFiles:
			sLibraries.append(sourceFile.libraryFullPath)
			sEntryNames.append(sourceFile.name)
			pEntryTypes.append(sourceFile.type.value)

			binSource = self.readSource(sourceFile.fullFilePath)
			sSource = binSource.decode("utf-8")
			sEntrySources.append(sSource)
			sEntryComments.append(self._getCommentFromSource(sSource))
			lEntrySourceLengths.append(len(sSource.encode('utf-16-le')))
			iNumberOfEntries += 1
		
		pLibraryNames = (win.LPWSTR * len(sLibraries))(*sLibraries)
		pEntryNames = (win.LPWSTR * len(sEntryNames))(*sEntryNames) #Entry Names
		otEntryTypes = (c.c_long * len(pEntryTypes))(*pEntryTypes)
		pComments = (win.LPWSTR * len(sEntryComments))(*sEntryComments)
		pEntrySyntaxBuffers = (win.LPWSTR * len(sEntrySources))(*sEntrySources)
		pEntrySyntaxBuffSizes = (c.c_long * len(lEntrySourceLengths))(*lEntrySourceLengths)
		
		return e.PBResult(self.dll.PBORCA_CompileEntryImportList(
			self.hSession,
			pLibraryNames,
			pEntryNames,
			otEntryTypes,
			pComments,
			pEntrySyntaxBuffers,
			pEntrySyntaxBuffSizes,
			iNumberOfEntries,
			fOrcaCompErr(self.cbkCompErr),
			0
		))
				
	@classmethod
	def _getCommentFromSource(cls, sSource):
		return re.search(r'(?<=(\r\n\$PBExportComments\$)).*?(?=(\r\n))|$', sSource[:500]).group(0)

	def sourceImport(self, pbl_path : Path, sSourceFile, errorList = []):
		self.returnList = errorList
		lpszLibraryName = (win.LPWSTR)(os.fspath(pbl_path))
		lpszEntryName = (win.LPWSTR)(ntpath.basename(sSourceFile)[:-4])
		otEntryType = (c.c_long)(e.PBSrcType.getType(sSourceFile).value)
		
		binSource = self.readSource(sSourceFile)
		lEntrySyntaxBuffSize = (c.c_long)(len(binSource))
		sSource = binSource.decode('utf-8-sig')

		lpszComments = (win.LPWSTR)(self._getCommentFromSource(sSource))
		lpszEntrySyntax = (win.LPWSTR)(sSource)
		
		return e.PBResult(self.dll.PBORCA_CompileEntryImport(
			self.hSession,
			lpszLibraryName,
			lpszEntryName,
			otEntryType,
			lpszComments,
			lpszEntrySyntax,
			lEntrySyntaxBuffSize,
			fOrcaCompErr(self.cbkCompErr),
			0
		))	
	
	def source_list(self, pbl_path : Path, srcList):
		self.returnList = srcList

		lpszLibName = (win.LPWSTR)(os.fspath(pbl_path))
		lpszLibComments = (win.LPWSTR)("".ljust(e.PBORCA_MSGBUFFER + 1))
		iCmntsBuffLen = (c.c_int)(e.PBORCA_MSGBUFFER + 1)
		return e.PBResult(self.dll.PBORCA_LibraryDirectory(
			self.hSession,
			lpszLibName,
			lpszLibComments,
			iCmntsBuffLen,
			fOrcaDirEntry(self.cbkDirEntry),
			0
		))

	def sourceExport(self, pbl_path : Path, sEntry, src : Src, eEntryType : e.PBSrcType, lEntrySize):
		'''
		sExportDir = self.workDir + "export\\" + sLib[:4]
		print("sourceExport.setConf", self.configure(exportFolder=sExportDir))
		Path(sExportDir).mkdir(parents = True, exist_ok = True)
		lpszLibraryName = (win.LPWSTR)(os.fspath(self.workDir.joinpath(sLib)))
		lpszEntryName = (win.LPWSTR)(sEntry)
		lpszExportBuffer = (win.LPWSTR)("".ljust(lEntrySize))
		plBufSize = (c.c_int)(lEntrySize)
		plReturnSize = (c.c_int)(0)
		ret = e.PBResult(self.dll.PBORCA_LibraryEntryExportEx(
			self.hSession,
			lpszLibraryName,
			lpszEntryName,
			eEntryType.value,
			lpszExportBuffer,
			plBufSize,
			c.byref(plReturnSize)
		))
		'''
		lpszLibraryName = (win.LPWSTR)(os.fspath(pbl_path))
		lpszEntryName = (win.LPWSTR)(sEntry)
		lpszExportBuffer = (win.LPWSTR)("".ljust(lEntrySize)) #todo: buffer size is often too small
		ret = e.PBResult(self.dll.PBORCA_LibraryEntryExport(
			self.hSession,
			lpszLibraryName,
			lpszEntryName,
			eEntryType.value,
			lpszExportBuffer,
			lEntrySize
		))
		src.source = lpszExportBuffer.value.rstrip()
		src.name = sEntry
		src.type = eEntryType
		src.library = pbl_path.name
		
		return ret
	
	def entryInfo(self, pbl_path : Path, sEntry, eEntryType : e.PBSrcType, infoList : list):
		pEntryInformationBlock = PBORCA_EntryInfo()
		ret = e.PBResult(self.dll.PBORCA_LibraryEntryInformation(
			self.hSession,
			(win.LPWSTR)(os.fspath(pbl_path)),
			(win.LPWSTR)(sEntry),
			eEntryType.value,
			c.byref(pEntryInformationBlock)
		))
		infoList.append(pEntryInformationBlock.szComments)
		infoList.append(pEntryInformationBlock.lCreateTime)
		infoList.append(pEntryInformationBlock.lObjectSize)
		infoList.append(pEntryInformationBlock.lSourceSize)
		return ret

	# Callback function for compilation errors while source imports
	def cbkCompErr(self, a, b):
		#print("orcaCompErr", a.contents.errorLevel, a.contents.msgNr, a.contents.msgText) 
		self.returnList.append([
			a.contents.errorLevel,
			a.contents.msgNr,
			a.contents.msgText,
			a.contents.colNr,
			a.contents.lineNr
		])
	
	# Callback function for source list result
	def cbkDirEntry(self, a, b):
		#print("orcaDirEntry", a.contents.lpszEntryName)
		self.returnList.append([
			a.contents.lpszEntryName, 
			e.PBSrcType(a.contents.otEntryType),
			a.contents.lEntrySize,
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
		('lEntrySize', c.c_long),
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
			
