!ifndef VERSION
  !define VERSION "dev"
!endif
!ifndef ARCH
  !define ARCH "amd64"
!endif
!ifndef OUTFILE
  !define OUTFILE "translate-mcp_setup.exe"
!endif
!ifndef BINARY
  !define BINARY "translate-mcp.exe"
!endif
!ifndef README_FILE
  !define README_FILE "README.md"
!endif
!ifndef LICENSE_FILE
  !define LICENSE_FILE "LICENSE"
!endif
!ifndef CONFIG_FILE
  !define CONFIG_FILE "config.example.yaml"
!endif

!define APP_NAME "translate-mcp"

OutFile "${OUTFILE}"
!include "MUI2.nsh"

Name "${APP_NAME} ${VERSION}"
!if "${ARCH}" == "amd64"
  InstallDir "$PROGRAMFILES64\${APP_NAME}"
!else
  InstallDir "$PROGRAMFILES\${APP_NAME}"
!endif
RequestExecutionLevel admin

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Section "Install"
  SetOutPath "$INSTDIR"
  File "${BINARY}"
  File "${README_FILE}"
  File "${LICENSE_FILE}"
  File "${CONFIG_FILE}"
  WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  Delete "$INSTDIR\translate-mcp.exe"
  Delete "$INSTDIR\README.md"
  Delete "$INSTDIR\LICENSE"
  Delete "$INSTDIR\config.example.yaml"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"
SectionEnd
