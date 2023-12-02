# 音訊檔案的管理

## 將音訊檔案的名稱翻譯為正確的名稱
.\dataPrep.exe -stagingDir D:\TW_SATI\staging -properName

## 為各個音訊檔案建立資料夾
.\dataPrep.exe -stagingDir D:\TW_SATI\staging -initMedia



# YouTube 上傳
__以下部分需要 google api 身份驗證設置__

請將 __client_secret.json__, __client_secret_drive.json__ 和 __.credentials__ 資料夾放在使用者的主資料夾(Home)中

## 上傳
.\youtube.exe -upload [檔案名稱]

## 上傳封面
.\youtube.exe -uploadCover [檔案名稱]

## 上傳字幕
.\youtube.exe -caption [檔案名稱]

## 隱藏視頻
.\youtube.exe -unlist [檔案名稱]

## 發布
.\youtube.exe -publish [檔案名稱]