Prompt: "请根据 Sealock Doc 的特性，使用 GORM 定义以下 PostgreSQL 模型，并处理好外键关系：

User: 基本信息及配额。

Repo: 资料库，需包含 OwnerID 和是否加密的标志。

Branch/Commit: 记录资料库的版本演进，指向特定的 Root Tree。

FileEntry: 记录文件名、大小、修改时间以及对应的 FileContentHash。

Block: 记录物理数据块的 Hash、大小及引用计数（用于清理孤儿块）。"