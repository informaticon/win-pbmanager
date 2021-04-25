import ntpath
from  pathlib import Path
import orca.enums as e
import os, sys
from tabulate import tabulate
from orca.interface import Orca
from orca.source import Src

def open_session():
	return Orca()

def export_pbl(orca_session : Orca, pbl_path : Path, pbl_export_folderpath : str) -> list[list[str]]:
	ret_code, pb_objects = orca_session.get_source_list(pbl_path)
	if ret_code != e.PBResult.PBORCA_OK:
		raise Exception("Couldn't get Source List")
	
	for pb_object in pb_objects:
		ret, srcInfo = orca_session.get_entry_info(pbl_path, pb_object[0], pb_object[1])
		if ret == e.PBResult.PBORCA_OK:
			pb_object.append(srcInfo[2])
			pb_object.append(srcInfo[3])
		else:
			pb_object.append(ret)
			pb_object.append(ret)
		
		srcExport = Src()
		ret, srcExport = orca_session.export_source(pbl_path, pb_object[0], pb_object[1], 2 + srcInfo[3])
		pb_object.append(ret.as_string())
		pb_object.append(srcExport)

		write_file(pbl_export_folderpath, srcExport.get_file_name(), srcExport.get_source())

	return pb_objects

def import_pbl(orca_session : Orca, pbl_path : Path, pbl_export_folderpath : Path):
	source_files = []
	returns = []

	for src_path in pbl_export_folderpath.glob('*.sr*'):
		source_files.append(
			Src(
				pbl_path = pbl_path,
				name = src_path.stem,
				own_path = src_path,
				src_type = e.PBSrcType.get_type(src_path),
		))

	print(orca_session.import_sources(source_files, returns))
	return returns

def write_file(pbl_export_folderpath : str, pbl_export_filename : str, source : str):
	Path(pbl_export_folderpath).mkdir(parents = True, exist_ok = True)
	if os.path.exists(os.path.join(pbl_export_folderpath, pbl_export_filename)):
  		os.remove(os.path.join(pbl_export_folderpath, pbl_export_filename))
	with open(os.path.join(pbl_export_folderpath, pbl_export_filename), 'x+', encoding="utf-8-sig", newline="", ) as f:
		f.write(source)
