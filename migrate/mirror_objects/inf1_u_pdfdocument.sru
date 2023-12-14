$PBExportHeader$inf1_u_pdfdocument.sru
forward
global type inf1_u_pdfdocument from pdfdocument
end type
end forward

global type inf1_u_pdfdocument from pdfdocument
end type
global inf1_u_pdfdocument inf1_u_pdfdocument

on inf1_u_pdfdocument.create
call super::create
TriggerEvent( this, "constructor" )
end on

on inf1_u_pdfdocument.destroy
TriggerEvent( this, "destructor" )
call super::destroy
end on

