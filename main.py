#!/usr/bin/python
from  pathlib import Path
import os, sys
from tabulate import tabulate
import pbhelper
import orca

if sys.argv[1] == '--fullexport' or sys.argv[1] == '--deltaexport':
	work_dir = Path(sys.argv[2])
	targets = pbhelper.find_targets(work_dir)

	for target in targets:
		orca_session = orca.open_session()
		for pbl_path in target.get_pbl_list():
			export_folder = pbl_path.own_path.parent.joinpath('src').joinpath(pbl_path.own_path.stem)

			if export_folder.exists():
				# if pbl was not changed since last export then skip it
				if sys.argv[1] == '--deltaexport':
					if export_folder.stat().st_mtime >= pbl_path.own_path.stat().st_mtime:
						continue
				
				# all files are deleted first because they could be deleted in the library
				for source_file in export_folder.iterdir():
					source_file.unlink()
			else:
				export_folder.mkdir(parents = True, exist_ok = True)
			
			print('begin with export of {}'.format(pbl_path))
			
			orca.export_pbl(orca_session, pbl_path.own_path, export_folder)

			# set modified date of export folder to a newer value than the the one of its library
			export_folder.touch()

	print('export finished')

elif sys.argv[1] == "--libimport":
	#contextmenu pbl -> Import
	library =  Path(sys.argv[2])
	if len(sys.argv) > 3:
		work_dir = Path(sys.argv[3])
	else:
		work_dir = library.parent
	target = pbhelper.find_targets(work_dir)[0]
	export_folder = library.parent.joinpath('src').joinpath(library.stem)
	
	po = orca.open_session()
	
	print("set_pbl_list", po.set_pbl_list(list(map(lambda t: t.get_path(), target.get_pbl_list()))))
	print("set_current_app", po.set_current_app(target.get_app_pbl_path(), target.get_app_name()))
	print("create", po.pbl_create(library, "Created by Import"))
	ret = orca.import_pbl(po, library, export_folder)
	print(tabulate(ret, headers=["State", "File", "error_list"]))
	
elif sys.argv[1] == "--libexport":
	#contextmenu pbl -> Export
	library =  Path(sys.argv[2])
	if len(sys.argv) > 3:
		work_dir = Path(sys.argv[3])
	else:
		work_dir = library.parent
	target = pbhelper.find_targets(work_dir)[0]
	export_folder = library.parent.joinpath('src').joinpath(library.stem)
	
	po = orca.open_session()
	
	print("set_pbl_list", po.set_pbl_list(list(map(lambda t: t.get_path(), target.get_pbl_list()))))
	print("set_current_app", po.set_current_app(target.get_app_pbl_path(), target.get_app_name()))
	print(tabulate(orca.export_pbl(po, library, export_folder), headers=["File", "Type", "Size", "ModifiedDateTime", "Comment", "ObjectSize", "SourceSize", "ExportResult", "Source"]))

else:
	raise NotImplementedError
