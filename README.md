# Gormzap

## 使用方式

```golang
    db.LogMode(true)
    //传入对象为任何原始的 *zap.Logger
    db.SetLogger(gormzap.New(logger.ZapLogger))

    // 默认输出为Debug 支持输出为Info (只支持 "debug"/"info")
    gormzap.DefaultOutputLevel = gormzap.OutputLevelInfo
```
