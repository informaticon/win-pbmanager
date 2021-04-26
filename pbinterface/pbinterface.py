#!/usr/bin/python
from  pathlib import Path
import os, sys
from tabulate import tabulate
from orca import enums as e
import pbhelper
import orca
import click

@click.group()
@click.option('--source-folder', '-s', 'source_folder', default='src/',
	help = 'Path to the folder where the source files are stored. Path can be absolute or relative to the folder of the pbl file which is processed.\nDefault ist "src/"',
	type = click.Path(file_okay=False, dir_okay=True)
)
@click.pass_context
def pbinterface(ctx={}, source_folder = 'src/'):
	ctx.ensure_object(dict)
	ctx.obj['source_folder'] = source_folder


@pbinterface.group(name="import")
@click.pass_context
def import_(ctx):
	"""Import source into PowerBuilder.
	
	You must specify the import subcommand (currently only pbl is supported)."""
	pass

@import_.command(name="pbl")
@click.argument('filepath',
	type=click.Path(file_okay=True, dir_okay=False, writable=True, readable=True)
)
@click.pass_context
def import_pbl(ctx, filepath=None):
	"""Import into FILEPATH.

	FILEPATH must be an existing pbl file.
	"""
	pbl_filepath =  Path(filepath)
	source_folder = Path(ctx.obj['source_folder'])
	
	pbt = pbhelper.find_targets(pbl_filepath.parent)[0]
	
	po = orca.open_session()
	
	print("set_pbl_list", po.set_pbl_list(list(map(lambda t: t.get_path(), pbt.get_pbl_list()))))
	print("set_current_app", po.set_current_app(pbt.get_app_pbl_path(), pbt.get_app_name()))
	print("create", po.pbl_create(pbl_filepath, "Created by Import"))
	ret = orca.import_pbl(po, pbl_filepath, get_export_folder(pbl_filepath, source_folder))
	print(tabulate(ret, headers=["State", "File", "error_list"]))


@pbinterface.group()
@click.pass_context
def export(ctx):
	"""Export source from PowerBuilder.
	
	You must specify the export subcommand.
	"""
	pass

@export.command(name='pbl')
@click.argument('filepath',
	type=click.Path(file_okay=True, dir_okay=False, writable=False, readable=True)
)
@click.pass_context
def export_pbl(ctx, filepath):
	"""Export source from pbl.
	
	FILEPATH must be an absolute filepath of a pbl file.
	"""
	pbl_filepath =  Path(filepath)
	source_folder = Path(ctx.obj['source_folder'])
	
	print(tabulate(do_export(orca.open_session(), pbl_filepath, get_export_folder(pbl_filepath, source_folder)), headers=["File", "Type", "Size", "ModifiedDateTime", "Comment", "ObjectSize", "SourceSize", "ExportResult", "Source"]))
	print('export finished')
	

@export.command(name='pbt')
@click.argument('filepath',
	type=click.Path(file_okay=True, dir_okay=False, writable=False, readable=True)
)
@click.option('--type', '-t', 'export_type', default='full',
	type=click.Choice(['full', 'delta']),
	help='Export strategy, choose from one of the following values:\n\nfull: Delete all existing sources and do a full export of all pbl files. [default]\n\ndelta: Export only the pbl files, if they have changed since the last export.'
)
@click.pass_context
def export_pbt(ctx, filepath, export_type):
	"""Export source from all pbl of one pbt.
	
	FILEPATH must be an absolute filepath of a pbt file.
	"""
	pbt = pbhelper.Pbt.parse(Path(filepath))
	source_folder = Path(ctx.obj['source_folder'])

	ret = {}
	for pbl in pbt.get_pbl_list():
		r = do_export(orca.open_session(), pbl.own_path, get_export_folder(pbl.own_path, source_folder), export_type)
		if r:
			ret[pbl] = r

	for o in ret:
		print(o)
		print(tabulate(ret[o], headers=["File", "Type", "Size", "ModifiedDateTime", "Comment", "ObjectSize", "SourceSize", "ExportResult", "Source"]))
	print('export finished')

@export.command(name='folder')
@click.option('--type', '-t', 'export_type', default='delta',
	type=click.Choice(['full', 'delta']),
	help='Export strategy, choose from one of the following values:\n\nfull: Delete all existing sources and do a full export of all pbl files.\n\ndelta: Export only the pbl files, if they have changed since the last export. [default]'
)
@click.argument('path',
	type=click.Path(file_okay=False, dir_okay=True, writable=False, readable=True)
)
@click.pass_context
def export_folder(ctx, export_type, path):
	"""Export source from all pbl of all pbt found in specific folder.
	
	FILEPATH must be an absolute path of an existing folder.
	"""
	pbt_list = pbhelper.find_targets(Path(path))
	source_folder = Path(ctx.obj['source_folder'])

	for pbt in pbt_list:
		print('begin with export of {}'.format(pbt))
		orca_session = orca.open_session()

		for pbl in pbt.get_pbl_list():
			do_export(orca_session, pbl.own_path, get_export_folder(pbl.own_path, source_folder), export_type)

	print('export finished')

def get_export_folder(pbl_filepath : Path, source_folder : Path):
	if source_folder.is_absolute():
		return source_folder.joinpath(pbl_filepath.stem)
	else:
		return pbl_filepath.parent.joinpath(source_folder).joinpath(pbl_filepath.stem)

def handle_pb_result(result):
	if result == e.PBResult.PBORCA_OK:
		return
	raise Exception(result.to_string())

def do_export(orca_session, pbl_filepath : Path, export_folder : Path, export_type='full'):
	if export_type == 'delta' and export_folder.exists():
		# if pbl was not changed since last export then skip it
		if export_folder.stat().st_mtime >= pbl_filepath.stat().st_mtime:
			return
	else:
		export_folder.mkdir(parents=True, exist_ok=True)
	
	# all files are deleted first because they could have been deleted in the library
	for source_file in export_folder.iterdir():
		source_file.unlink()
	
	print('   begin with export of {}'.format(pbl_filepath))
	
	orca.export_pbl(orca_session, pbl_filepath, export_folder)

	# set modified date of export folder to a newer value than the the one of its library
	export_folder.touch()
	
	return orca.export_pbl(orca_session, pbl_filepath, export_folder)

if __name__ == '__main__':
	pbinterface()
