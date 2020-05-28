
## Introductiuon

Wframe is simple & lighting HTTP server framework for middleware developement. 
it origin from 'Qujie Verframe ([Qujie Tech. Ltd.](https://www.qujietech.com/))'
framework in some inner porject.

Wframe provide a HTTP RPC framework with a clear lifecircle. it based native
'net/http' library int golang, compatible native golang HTTP handle object.

Framework feature:

- new style request handler with clear lifecircle
- run multiple instance with different basepath and different configure
- config you server instance use YAML file
- optional RESTful style
- modularize integration

## Lifecricle


```
+----------------------------------------------------------+
|                        [QInstance]                       |
+-------------------------------+--------------------------+
|            [QHandle]          |          [QEnv]          |
| * InitHandler                 | * InitEnv                |
+ ............................. + ........................ |
|                         (HTTP Request)                   |
+       ....................    + ........................ |
|      |                    |   | * ReqForEnv              |
|  +-->| * BeginSession     |   +--------------------------+
|  |   |                    |   |    request env object    |
+- |   +--------------------+---+--------------------------+
|  |                      [QSession]                       |
|  | * EnterServer                                         |
|  |   ....................                                |
|  +--| * Inner redirect   |                               |
|      ....................                                |
+ ........................................................ +
| * BeginResponse                                          |
+ ........................................................ +
|                     (HTTP Response)                      |
+ ........................................................ +
| * WriteResponse                                          |
+----------------------------------------------------------+
|                      (End Response)                      |
+----------------------------------------------------------+
```

