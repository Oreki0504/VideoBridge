#ifndef PayloadDir
  #error PayloadDir must point to the prepared installer payload
#endif
#ifndef AppVersion
  #error AppVersion is required
#endif
#ifndef HostFile
  #error HostFile is required
#endif
[Setup]
AppId={{8A8DD097-86D8-4ED0-9C18-A83A6F23DA15}
AppName=VideoBridge
AppVersion={#AppVersion}
AppPublisher=VideoBridge
DefaultDirName={localappdata}\VideoBridge
DefaultGroupName=VideoBridge
DisableDirPage=yes
DisableProgramGroupPage=yes
AllowNoIcons=yes
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
WizardStyle=modern
Compression=lzma2
SolidCompression=yes
OutputDir=..\dist
OutputBaseFilename=VideoBridge-{#AppVersion}-Setup
UninstallDisplayName=VideoBridge
UninstallDisplayIcon={app}\unins000.exe
CloseApplications=no
RestartApplications=no
SetupMutex=VideoBridgeSetup
UsePreviousTasks=yes
DisableWelcomePage=no

[Languages]
Name: "chinesesimp"; MessagesFile: "compiler:Languages\ChineseSimplified.isl"

[Messages]
WelcomeLabel2=此向导将安装 VideoBridge 的浏览器扩展和本地下载程序。%n%n所有程序组件集中保存在当前用户的 VideoBridge 目录，不需要管理员权限。已有下载目录和工具配置会保留。%n%n安装后只需在 Chrome / Edge 中手动加载扩展一次。
FinishedHeadingLabel=VideoBridge 已安装
FinishedLabel=请查看安装说明，在浏览器中加载固定目录里的扩展。%n%n如果之前使用过解压版，请移除旧扩展，再加载新目录。下载文件和本地配置会保留。

[Tasks]
Name: "chrome"; Description: "连接 Chrome"; Flags: checkedonce
Name: "edge"; Description: "连接 Microsoft Edge"; Flags: unchecked

[Files]
Source: "{#PayloadDir}\extension\*"; DestDir: "{app}\extension"; Flags: ignoreversion recursesubdirs createallsubdirs
Source: "{#PayloadDir}\host\{#HostFile}"; DestDir: "{app}\host"; Flags: ignoreversion onlyifdoesntexist
Source: "{#PayloadDir}\host\build-info.json"; DestDir: "{app}\host"; Flags: ignoreversion
Source: "{#PayloadDir}\setup\*"; DestDir: "{app}\setup"; Flags: ignoreversion

[Icons]
Name: "{group}\安装说明"; Filename: "{app}\setup\welcome.html"
Name: "{group}\扩展文件夹"; Filename: "{win}\explorer.exe"; Parameters: """{app}\extension"""
Name: "{group}\卸载 VideoBridge"; Filename: "{uninstallexe}"

[Run]
Filename: "{app}\setup\welcome.html"; Description: "查看浏览器扩展安装说明"; Flags: shellexec postinstall skipifsilent nowait
Filename: "{win}\explorer.exe"; Parameters: """{app}\extension"""; Description: "打开扩展文件夹"; Flags: postinstall skipifsilent nowait unchecked

[UninstallRun]
Filename: "{sys}\WindowsPowerShell\v1.0\powershell.exe"; Parameters: "-NoProfile -NonInteractive -ExecutionPolicy Bypass -File ""{app}\setup\Unregister-Installation.ps1"" -Root ""{app}"""; Flags: runhidden waituntilterminated; RunOnceId: "UnregisterNativeHost"

[Code]
var ConfigurationFailed: Boolean;

function NextButtonClick(CurPageID: Integer): Boolean;
begin
  Result := True;
  if (CurPageID = wpSelectTasks) and not (WizardIsTaskSelected('chrome') or WizardIsTaskSelected('edge')) then begin
    MsgBox('请至少选择一个要连接的浏览器。', mbError, MB_OK);
    Result := False;
  end;
end;

procedure SetupLog(const S: String; const Error, FirstLine: Boolean);
begin
  Log(S);
  if Pos('…', S) > 0 then WizardForm.StatusLabel.Caption := S;
end;

procedure CurStepChanged(CurStep: TSetupStep);
var Browser, Params: String; ExitCode: Integer;
begin
  if CurStep = ssPostInstall then begin
    Browser := 'Chrome';
    if WizardIsTaskSelected('edge') then begin
      if WizardIsTaskSelected('chrome') then Browser := 'Both' else Browser := 'Edge';
    end;
    Params := '-NoProfile -NonInteractive -ExecutionPolicy Bypass -File "' + ExpandConstant('{app}\setup\Configure-Installation.ps1') + '" -Root "' + ExpandConstant('{app}') + '" -Browser ' + Browser;
    if ExpandConstant('{param:SKIPDEPENDENCIES|0}') = '1' then Params := Params + ' -SkipDependencies';
    WizardForm.StatusLabel.Caption := '正在配置浏览器连接并检查依赖；首次下载可能需要几分钟…';
    try
      ConfigurationFailed := not ExecAndLogOutput(ExpandConstant('{sys}\WindowsPowerShell\v1.0\powershell.exe'), Params, '', SW_HIDE, ewWaitUntilTerminated, ExitCode, @SetupLog);
      ConfigurationFailed := ConfigurationFailed or (ExitCode <> 0);
    except
      Log(GetExceptionMessage);
      ConfigurationFailed := True;
    end;
    if ConfigurationFailed then begin
      WizardForm.FinishedLabel.Caption := '程序文件已安装，但本地配置或依赖安装未完成。请检查网络后重新运行安装程序，或在扩展中点击“检测并补齐依赖”。';
      if not WizardSilent then MsgBox('本地配置或依赖安装未完成。程序文件已保留，可重新运行安装程序继续。', mbError, MB_OK);
    end;
  end;
end;

function GetCustomSetupExitCode: Integer;
begin
  if ConfigurationFailed then Result := 1 else Result := 0;
end;

procedure CurPageChanged(CurPageID: Integer);
begin
  if (CurPageID = wpFinished) and ConfigurationFailed then begin
    WizardForm.FinishedHeadingLabel.Caption := '还需完成本地配置';
    WizardForm.FinishedLabel.Caption := '程序文件已安装，但本地配置或依赖安装未完成。请检查网络后重新运行安装程序，或在扩展中点击“检测并补齐依赖”。';
  end;
end;
