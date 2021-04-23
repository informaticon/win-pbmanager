import ntpath
from  pathlib import Path
import orca.enums as e
import os, sys
from tabulate import tabulate
from orca.interface import Orca
from orca.source import Src

# PBL Import single File
'''
print("libCreate", po.libraryCreate("exf1.pbl", "Er geht"))
ret = po.sourceImport("exf1.pbl", "C:\\a3\\gs_exf\\d\\exf1_u_exf_blob.sru")
print("sourceImport", e.PBResult.toString(ret[0]))
print(tabulate(ret[1], headers=["State", "File"]))
'''
def open_session(pbt_path : Path):
	if pbt_path.is_file():
		pbt_path = pbt_path.parent
	return Orca(pbt_path)

def export_pbl(orca_session : Orca, pbl_path : Path, pbl_export_folderpath : str) -> list[list[str]]:
	pb_objects : list[list[str]] = []
	if orca_session.source_list(pbl_path, pb_objects) != e.PBResult.PBORCA_OK:
		raise Exception("Couldn't get Source List")
	
	for pb_object in pb_objects:
		srcInfo = []
		ret = orca_session.entryInfo(pbl_path, pb_object[0], pb_object[1], srcInfo)
		if ret == e.PBResult.PBORCA_OK:
			pb_object.append(srcInfo[2])
			pb_object.append(srcInfo[3])
		else:
			pb_object.append(ret)
			pb_object.append(ret)
		
		srcExport = Src()
		pb_object.append(orca_session.sourceExport(pbl_path, pb_object[0], srcExport, pb_object[1], 2 + srcInfo[3]).asString())
		pb_object.append(srcExport)

		write_file(pbl_export_folderpath, srcExport.getFileName(), srcExport.getSource())

	return pb_objects

#example for single elements import
'''
def import_pbl(pbl_filename, pbl_export_folderpath):
	files = []
	returns = []

	for file in os.listdir(pbl_export_folderpath):
		errorList = []
		fileFullPath = os.path.join(pbl_export_folderpath, file)
		files.append(fileFullPath)
		returns.append([orca_session.sourceImport(pbl_filename, fileFullPath, errorList), file, errorList])

	return returns
'''

#example for list-import
def import_pbl(orca_session : Orca, pbl_filename : str, pbl_export_folderpath : str):
	sourceFiles = []
	returns = []

	for file in os.listdir(pbl_export_folderpath):
		sourceFile = Src()
		sourceFile.libraryFullPath = os.fspath(orca_session.workDir.joinpath(pbl_filename))
		sourceFile.name = Path(file).stem
		sourceFile.fullFilePath = os.fspath(Path(pbl_export_folderpath).joinpath(file))
		sourceFile.type = e.PBSrcType.getType(file)
		sourceFiles.append(sourceFile)

	print(orca_session.sourceImportBatch(sourceFiles, returns))
	return returns

def write_file(pbl_export_folderpath : str, pbl_export_filename : str, source : str):
	Path(pbl_export_folderpath).mkdir(parents = True, exist_ok = True)
	if os.path.exists(os.path.join(pbl_export_folderpath, pbl_export_filename)):
  		os.remove(os.path.join(pbl_export_folderpath, pbl_export_filename))
	with open(os.path.join(pbl_export_folderpath, pbl_export_filename), 'x+', encoding="utf-8-sig", newline="", ) as f:
		f.write(source)

def get_library_list(sTarget : str):
	libraries = []
	with open(sTarget, "r+") as f:
		for line in f:
			if line.startswith ('LibList') or line.startswith ('liblist'):
				libraries = line.split('"')[1::2][0].split(';')
	return libraries
