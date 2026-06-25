# Prompts 分层说明

`prompts/` 是这个项目的规则中心。

- `tools/` 负责执行
- `prompts/` 负责定义怎么生成、怎么纠偏、怎么自检
- Prompt 用来定义生成方向，不用来写固定答案

## 分层总览

这个目录下的 Prompt 按职责分成 5 层。

## 文件分组

如果只想快速判断“该看哪份 Prompt”，可以先按下面 4 组理解：

### A. 输入与入口

- `intake.md`

### B. 角色本体

- `persona_analyzer.md`
- `persona_builder.md`
- `lore_analyzer.md`
- `lore_builder.md`

### C. 关系与表现增强

- `relationship_builder.md`
- `custom_builder.md`
- `relationship_scene_builder.md`

### D. 纠偏、校验与评估

- `merger.md`
- `correction_handler.md`
- `validator.md`
- `eval_guide.md`

### 1. 输入层

- `intake.md`
  生成前收集角色名、页面 URL、关系身份、Skill 偏向

### 2. 固定数据层

- `persona_analyzer.md`
  从网页资料里提炼角色本体的人格、说话方式、情绪机制、互动倾向
- `persona_builder.md`
  把人格信息整理成 `persona.md`
- `lore_analyzer.md`
  从网页资料里提炼设定事实
- `lore_builder.md`
  把事实信息整理成 `lore.md`

固定数据层原则：

- 只提炼网页有依据的内容
- 页面没写就不要补成事实
- 朋友、恋人、同事等关系身份不参与这一层

### 3. 后置增强层

- `relationship_builder.md`
  定义关系身份下的互动位置、距离感、边界和情绪偏移
- `custom_builder.md`
  定义长期稳定的表达规律，例如高频互动、语言纹理、降级策略、长期陪伴感
- `relationship_scene_builder.md`
  只在“身份没问题，但真实对话感不够”时用于专项校准

后置增强层原则：

- 只能增强表现，不改写角色设定
- 所有规则都必须服从 `persona.md` 和 `lore.md`
- `relationship_scene_builder.md` 更偏调优辅助，不是常驻脚本模板

### 4. 修正层

- `merger.md`
  新资料或补充设定进来时，先判断属于哪一层，再做增量合并
- `correction_handler.md`
  用户指出 OOC 或设定错误时，先判断问题层级，再决定修正位置

### 5. 校验层

- `validator.md`
  检查分层是否被破坏
- `eval_guide.md`
  评估 Claude 实测结果，判断问题更接近角色层、身份层、场景层还是模板化问题

## 推荐使用顺序

1. `intake.md`
2. `persona_analyzer.md` / `persona_builder.md`
3. `lore_analyzer.md` / `lore_builder.md`
4. `relationship_builder.md`
5. `custom_builder.md`
6. 需要专项校准时再用 `relationship_scene_builder.md`
7. `validator.md`
8. 实测后用 `eval_guide.md`
9. 有问题时回到 `merger.md` / `correction_handler.md`

## 按问题找 Prompt

- 想补“她是谁”：
  先看 `persona_*` 和 `lore_*`
- 想补“她以什么身份对你说话”：
  先看 `relationship_builder.md`
- 想补“某种场景下怎么更像真人对话”：
  先看 `relationship_scene_builder.md`
- 想补“整体为什么太模板、太硬、太像流程”：
  先看 `custom_builder.md`
- 想判断“到底该改哪一层”：
  先看 `validator.md`、`merger.md`、`correction_handler.md`

## 情绪表达怎么分层

- `persona_*`
  负责角色本体的情绪机制
- `relationship_builder.md`
  负责关系身份下的情绪偏移
- `relationship_scene_builder.md`
  负责具体场景里的情绪落地
- `custom_builder.md`
  负责长期稳定的表达节奏和收束习惯

## 一句话原则

- 网页数据负责定角色
- 关系身份负责定互动位置
- 自定义补充负责让角色更像真人对话
- 修正和校验负责保证这些层不混掉
