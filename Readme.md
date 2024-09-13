# **1. 引言**

我们将深入解析 Informer 机制，揭示其工作原理、核心组件及其在提升资源变更处理效率方面的优势。

# **2. 为什么需要Informer**

在 Kubernetes 中，直接使用 `list` 和 `watch` API 可以实现资源的监控和管理，但这种方法在实际应用中存在一些局限性。随着集群规模的扩大和资源数量的增加，简单地依赖 `list` 和 `watch` 可能会带来性能瓶颈和管理上的复杂性。因此，Kubernetes 引入了 Informer 机制，以更高效、可靠地处理资源的变更和状态管理

# **3. Informer 的基本概念**

- **Informer 是什么**

在 Kubernetes 中，Informer 是一种用于高效地监听和缓存 Kubernetes 资源对象的机制。它通过与 API Server 通信，接收并处理资源对象的增删改事件，将这些变化通知给注册的处理程序。Informer 在 Kubernetes 客户端库（client-go）中实现，是构建控制器的核心组件之一。

- **Informer 的作用**


- **减少 API Server 的压力**
    - Informer 使用 List-Watch 机制，首先通过 List 操作获取所有资源对象的当前状态，然后通过 Watch 操作持续监听资源的变化。这样，Informer 可以及时捕获资源的变化事件，并将其应用到本地缓存中，减少了频繁的 API 请求。
- **提高资源变更的处理效率**
    - Informer 提供了事件驱动的编程模型，当资源对象发生变化时，Informer 会生成相应的事件并通知注册的事件处理程序（Handler）。这种模型可以确保资源的变化被及时处理，减少了延迟。
    - **提供资源缓存**
    - Informer 维护一个本地缓存，将获取到的资源对象存储在内存中。这样，控制器可以直接从本地缓存中获取资源状态，提高了访问速度和效率，同时减少了对 API Server 的访问频率。
- **Informer 与 List & Watch 的关系**
    - 简单来说`Informer` 是对 `List & Watch` 的封装和扩展。它不仅负责调用 `List & Watch` 来获取和监控资源，还会将资源的最新状态缓存起来，并提供事件驱动的机制来处理资源变化。
    - `List & Watch` 是基础，而 `Informer` 是为了让开发者更容易且更高效地使用 `List & Watch` 机制。
    - 在开发自定义控制器或 Operator 时，`Informer` 提供了强大的工具，开发者无需直接处理低级别的 `List & Watch` 逻辑，只需专注于业务逻辑的实现即可。

# **4. Informer 的工作原理**


1. **SharedInformer**

`SharedInformer` 是 Informer 体系中的核心组件，负责管理所有其他模块的协作。它的主要职责是：

- **资源对象的监听和分发**：`SharedInformer` 从 API Server 中接收资源对象的变化事件，并将这些事件广播给所有注册了的事件处理器。
- **数据缓存**：`SharedInformer` 通过内置的 `Indexer` 模块缓存资源对象数据，使得其他组件能够快速访问最新的资源状态，而无需频繁访问 API Server。
- **同步机制**：它会周期性地同步资源对象的状态，确保本地缓存与集群状态的一致性。
1. **Indexer**

`Indexer` 是 Informer 内部用于缓存资源对象的组件，其职责包括：

- **数据存储**：`Indexer` 将从 API Server 接收到的资源对象存储在本地内存中，作为缓存使用。
- **索引功能**：除了简单地存储对象外，`Indexer` 还可以基于指定的字段对对象进行索引，以支持更高效的数据查询。
- **数据检索**：其他组件（如 `Lister` 和事件处理器）可以通过 `Indexer` 快速检索所需的资源对象，减少直接访问 API Server 的频率。
1. **Lister**

`Lister` 是一个查询接口，负责从 `Indexer` 中检索资源对象，其职责包括：

- **提供查询接口**：`Lister` 提供了一组方法，可以方便地查询缓存中的资源对象。常见的操作包括按名称、标签选择器或索引键来获取资源对象。
- **减少 API Server 负载**：通过 `Lister`，可以减少对 API Server 的直接请求，从而降低集群的负载和延迟。
1. **Reflector**

`Reflector` 是 Informer 的数据同步模块，负责从 API Server 中同步资源对象数据到本地缓存中。其主要职责包括：

- **Watch API 的管理**：`Reflector` 使用 Kubernetes 的 Watch API 来监控资源对象的变化事件（如添加、更新、删除），并将这些事件发送给 `SharedInformer` 处理。
- **定期重新同步**：在使用 Watch API 进行增量更新的同时，`Reflector` 还会定期执行全量同步操作，确保本地缓存与集群状态的完全一致。
- **处理事件**：它会根据接收到的事件更新 `Indexer` 中的缓存数据。
1. **Workqueue**

`Workqueue` 是 Informer 体系中的任务队列，用于处理资源对象变化的相关事件。其职责包括：

- **速率限制：** `Workqueue` 提供了对任务处理的速率限制功能，这在高负载情况下尤为重要。限流机制可以防止系统过载，确保任务处理的平稳性和可靠性。
- **事件去重**：`Workqueue` 会确保相同的资源对象不会被重复处理，从而避免资源浪费和处理冲突。
- **任务调度**：`Workqueue` 可以控制并发的任务数，确保在高负载情况下，系统能够稳定运行。
- **延迟重试**：对于失败的任务，`Workqueue` 支持延迟重试机制，允许在后续重新尝试处理。
1. **Informer 模块的协作流程**
- **事件获取与同步**：`Reflector` 启动后，通过 Kubernetes API Server 的 Watch API 开始监听指定资源的变化事件。初次同步时，它会先执行一个 List 操作获取当前所有的资源对象，并将这些对象缓存到 `Indexer` 中。
- **事件分发与缓存更新**：当 `Reflector` 捕获到资源对象的变化事件后，会将这些事件发送给 `SharedInformer`。`SharedInformer` 会根据事件类型（添加、更新、删除）来更新 `Indexer` 中的缓存。
- **任务入队与去重**：当缓存更新后，`SharedInformer` 会将处理任务推送到 `Workqueue` 中。`Workqueue` 会对任务进行去重，确保同一个资源对象不会被多次处理。
- **任务执行与数据查询**：消费任务时，处理器会通过 `Lister` 从 `Indexer` 中获取最新的资源对象数据，进行相应的处理。
- **完成与重新入队**：任务处理完成后，如果需要重试，可以重新将任务放入 `Workqueue` 中。否则，任务将标记为完成。