
import orca.enums as e
from pathlib import Path
class Src:
	def __init__(self, source = None, src_type : e.PBSrcType = None, name = None, own_path : Path = None, pbl_path : Path = None):
		self.source = source
		self.src_type = src_type
		self.name =  name
		self.own_path = own_path
		self.pbl_path = pbl_path

	def get_file_name(self):
		return self.name + self.src_type.get_file_ending()

	def get_source(self):
		return self.source
