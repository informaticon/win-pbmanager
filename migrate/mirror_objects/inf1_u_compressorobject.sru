$PBExportHeader$inf1_u_compressorobject.sru
forward
global type inf1_u_compressorobject from compressorobject
end type
end forward

global type inf1_u_compressorobject from compressorobject
end type
global inf1_u_compressorobject inf1_u_compressorobject

on inf1_u_compressorobject.create
call super::create
TriggerEvent( this, "constructor" )
end on

on inf1_u_compressorobject.destroy
TriggerEvent( this, "destructor" )
call super::destroy
end on

