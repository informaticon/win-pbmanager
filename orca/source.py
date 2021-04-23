
import orca.enums as e

class Src:
	def __init__(self):
		self.source : str = None
		self.type : e.PBSrcType = None
		self.name : str =  None
		self.fullFilePath : str = None
		self.libraryFullPath : str = None

	def getFileName(self):
		return self.name + self.type.getFileEnding()

	def getSource(self):
		return self.source
