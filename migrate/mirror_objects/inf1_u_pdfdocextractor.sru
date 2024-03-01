$PBExportHeader$inf1_u_pdfdocextractor.sru
forward
global type inf1_u_pdfdocextractor from pdfdocextractor
end type
end forward

global type inf1_u_pdfdocextractor from pdfdocextractor
end type
global inf1_u_pdfdocextractor inf1_u_pdfdocextractor

on inf1_u_pdfdocextractor.create
call super::create
TriggerEvent( this, "constructor" )
end on

on inf1_u_pdfdocextractor.destroy
TriggerEvent( this, "destructor" )
call super::destroy
end on

