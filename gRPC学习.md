## gRPC学习

### gRPC概述

- 概览

    gRPC是Google开发并开源的一套语言中立的RPC框架

- 特点

    1. 语言中立
    2. 基于IDL文件定义服务，通过proto3工具生成指定语言的数据结构，服务端接口以及客户端Stub
    3. 通信协议基于标准的HTTP/2设计，支持双向流，消息头压缩，单TCP的多路复用，服务端推送等特性，可以在移动端设备上更加省电和节省流量
    4. 序列化支持Protocol Buffer 和JSON协议，该协议是一种语言无关的高性能序列化框架

### 服务端创建流程

gRPC服务端java版本的实现使用了Build模式，对底层服务绑定，transportServer和NettyServer的创建和实例化都做了封装和屏蔽，让服务调用者不用关心gRPC的调用细节

整体流程分为3步：

1. 创建Netty HTTP/2服务端
2. 将需要的服务端接口实现类注册到内容的Registy中，RPC调用时，可以根据RPC请求消息中的服务定义信息查询到服务接口实现类
3. 创建gRPC Server，它是gRPC服务端的抽象，聚合了各种Listener，用于RPC消息的统一调度和处理

![img](https://gitee.com/Euraxluo/images/raw/master/picgo/c64c0e8e97711dc62e866861cd5c2e37.png)

关键流程分析：

1. **NettyServer 实例创建**：

    首先需要初始化NettyServer，它是gRPC基于Netty4.1 HTTP/2协议栈上封装的HTTP/2 服务端。NettyServer构建完成后，监听指定的Socket地址，即可实现基于HTTP/2协议的消息头接入

2. **绑定IDL定义的服务接口实现类:**

    gRPC与其他很多RPC框架不同的是，服务接口实现类的调用并不是通过动态代理和反射机制，而是通过proto工具生成代码，在服务端启动时，将服务接口实现类实例注册到gRPC的内部服务注册中心上。当请求消息接入后，可以根据服务名和方法名，直接调用启动时注册的服务实例，而不需要通过反射的方式进行调用，性能更好

3. **gRPC服务实例（ServerImpl构建）:**

    ServerImpl负责整个gRPC服务端消息的调度和处理，创建ServerImpl实例过程中，会对服务端依赖的对象进行初始化，例如Netty的线程池资源，gRPC的线程池，内部的服务注册类（InternalHandlerRegitry）等，ServerImpl初始化完成后，就可以调用NettyServer的start方法启动HTTP/2服务端，接收gRPC客户端的服务调用请求

### 服务端service调用流程

gRPC的客户端请求消息由Netty Http2ConnectionHandler接入，由gRPC负责将PB或者JSON消息反序列化为POJO对象，然后通过服务定义查询到该消息对应的接口实力，发起本地java接口调用。调用完成后，将响应消息序列化为对应和格式，通过HTTP2 Frame发送回客户端

整体流程分为4步：

1. gRPC请求消息接入
2. gRPC消息头和消息体处理
3. 内部的服务路由和调用
4. 响应消息发送

关键流程分析

1. **gRPC请求消息接入**

    Netty通过底层的HTTP/2协议栈，通过Http2ConnectionHandler，实现了HTTP/2消息的统一接入和处理。gRPC通过注册Http2FrameListener监听器，回调接收HTTP2协议的消息数据。gRPC 通过 FrameListener 重载 Http2FrameListener 的 onDataRead、onHeadersRead 等方法，将Netty的HTTP/2消息转发到gRPC的NettyServerHandler中，实现基于HTTP/2的RPC请求消息接入

    ![img](https://gitee.com/Euraxluo/images/raw/master/picgo/b269a81ef5012a8ed5409e97c071eecf.png)

2. **gRPC 消息头处理**

   通过NettyServerHandler的onHeadersRead()方法，实现对gRPC消息头和消息体的处理，流程如下
   
   1.对HTTP Header的Content-Type校验，此处必须是"application/grpc"
   
   2.从HTTP Header的URL中提取接口名和方法名
   
   3.将Netty的HTTP Header转换为gRPC内部的Metadata，Metadata内部维护了一个键值对的二维数组namesAndValues
   
   4.创建NettyServerStream对象，它持有了Sink和TransportState类，负责将消息封装为GrpcFrameCommand，与底层Netty进行交互，实现协议消息的处理
   
   5.创建NettyServerStreatm之后，触发erverTransportListener的streamCreated方法，完成消息上下文和gRPC业务监听器的创建
   
   6.gRPC上下文的创建，CancellableContext.CancellationListener的cancel方法，发送CancelServerStreamCommand指令
   
   7.JumpToApplicationThreadServerStreamListener 的创建，从 ServerStream 跳转到应用线程中进行服务调用，gRPC 服务端的接口调用主要通过 JumpToApplicationThreadServerStreamListener 的 messageRead 和 halfClosed 方法完成
   
   8.将 NettyServerStream 的 TransportState 缓存到 Netty 的 Http2Stream 中，当处理请求消息体时，可以根据 streamId 获取到 Http2Stream，进而根据“streamKey”还原 NettyServerStream 的 TransportState，进行后续处理
   
   ![img](https://gitee.com/Euraxluo/images/raw/master/picgo/125c42c9d4f333e23b9896f478517099.png)

3. **gRPC消息体处理**

    通过NettyServerHandler的onDataRead()方法，实现对gRPC消息头和消息体的处理，流程如下

    ![img](https://gitee.com/Euraxluo/images/raw/master/picgo/20c7a0f2ca1c9b801a406777283cb99a.png)

    关键步骤:

    1.gRPC NettyServerHandler在处理完消息头之后需要缓存上下文，因为处理消息体时也需要使用

    2.onDataRead和onHeadersRead方法都是由NIO线程进行调度，同时在执行时，会采用并行+交叉串行的方式运行

4. **内部的服务路由和调用**

    内部的服务路由和调用，主要包括如下步骤

    1.将请求消息体反序列化为JAVA的POJO对象，即IDL中定义的请求参数对象

    2.根据请求消息头中的方法名到注册中心查询对象的服务定义信息

    3.通过JAVA本地接口调用方式，调用服务端启动时注册对的IDL接口实现类

    流程如下：

    ![img](https://gitee.com/Euraxluo/images/raw/master/picgo/808ed4d507d2f72624f80457b2d3ca4d.png)

    关键步骤：

    1. 解码：对HTTP/2 Body进行应用层解码，转换为服务端接口的请求参数，解码的关键是调用requestMarshaller.patse(),将PB码流转换为JAVA对象
    2. 路由：根据URL中的方法名从内部服务注册中心查询到对应的服务实例，路由的关键是调用registry.lookupMethod(methodName)获取到ServerMethodDefinition对象
    3. 调用：调用服务端接口实现类的指定方法，实现RPC调用，与一些RPC框架不同的是，此处调用是Java本地接口，不是反射调用，因此性能更优，实现关键是UnaryRequestMethod.invoke(request,responseObserver)方法

5. **响应消息发送**

    响应消息的发送由StreamObserver的onNext触发，流程如下

    1.分别发送gRPC HTTP/2响应消息头和消息体，由NettyServerStream的Sink将响应消息封装为SendResponseHeadersCommand和SendGrpcFrameCommand，加入到WriteQueue中

    2.WriteQueue通过Netty的NioEventLoop线程进行消息处理，NioEventLoop将SendResponseHeadersCommand和SendGrpcFrameCommand写入到Netty的Channel中，进而触发DefaultChannelPipeline的write(Object msg,ChannelPipeline)操作

    3.响应消息通过ChannelPipeline职责链进行调度，触发NettyServerHandler的SendResponseHeaders和sendGrpcFrame方法，调用Http2ConnextionEncoder的witeHeaders和writeData方法，将响应消息通过Netty的HTTP/2协议栈发送给客户端

    ![img](https://gitee.com/Euraxluo/images/raw/master/picgo/77aedcd98c5910cad5f8d3e50cddc2a6.png)

    

### 服务端线程模型

#### BIO

调度模型：

- 服务端监听线程Acceptor负责客户端连接的接入，每当有新的客户端接入，就会创建一个新的I/O线程负责处理Socket
- 客户端请求消息的读取和应答的发送，都由I/O线程负责
- 除了I/O读写操作，默认情况下业务的逻辑处理，例如DB操作，也都在I/O线程处理
- I/O操作采用同步阻塞操作，读写没有完成，I/O线程会同步阻塞

存在的问题：

1. 性能问题：一连接一线程模型导致服务端的并发接入数和系统吞吐量受到极大的限制
2. 可靠性问题：由于I/O操作采用同步阻塞模式，当网络拥塞或者通信对端处理缓慢会导致I/O线程被挂起，阻塞时间无法预测
3. 可维护性问题：I/O线程数无法有效控制、资源无法有效共享（多线程并发问题）、系统可维护性差

优化方向：

- 池化：

    ​		为了解决同步阻塞I/O面临的一个链路需要一个线程处理的问题。会使用线程池来处理客户端的请求接入，形成客户端个数“M”与线程池最大线程数“N”的比例关系，其中M可以远远大于N，通过线程池可以灵活的调配线程资源，设置线程的最大值，防止由于海量并发接入导致线程耗尽。

    ​		通过使用线程池，避免了为每个请求都创建一个独立线程造成的线程资源耗尽问题。但是由于底层依然是同步阻塞模型，线程阻塞时间依然取决于对方的处理速度，优化之后的BIO线程模型依然无法从根本上解决性能线性扩展问题

BIO线程模型如下图所示：

![](https://gitee.com/Euraxluo/images/raw/master/picgo/6da1ab2c9d34a3660374a8a997ac03ca.png)

#### NIO

NIO在2002年以JSR-51的身份随JDK发布。

Selector会不断轮询注册在其上的Channel，如果某个Channel上面有新的TCP连接接入】读和写事件，这个Channel就会处于就绪状态，会被Selector轮询出来，然后通过SelectionKey可以获取就绪Channel集合，进行后续的I/O操作

通常一个I/O线程会聚合一个Selector，一个Selector可以同时注册N个Channel，这样单个I/O线程都可以同时并发处理多个客户端连接。另外，由于I/O操作是非阻塞的，因此也不会受限于网络速度和对方端点的处理时延，可靠性和效率大大提升

NIO线程模型（Reactor模式）如下图所示:

![](https://gitee.com/Euraxluo/images/raw/master/picgo/b5233ba74ab40a0ad9f4c7822164795f.png)



#### gRPC线程模型

##### gRPC服务端线程模型

java语言实现的gRPC由Netty线程和gRPC应用线程组成，如下图所示：

![](https://gitee.com/Euraxluo/images/raw/master/picgo/7b75fb1c58e0bee27cddc8b3f3e843b3.png)

1. **Netty Server 线程模型**

    工作流程如下：

    1. 从主线程池（bossGroup）中随机选择一个Reactor线程作为Acceptor线程，用于绑定监听端口，接收客户端连接
    2. Acceptor线程接收客户端连接请求后创建新的SocketChannel，将其注册到主线程池（bossGroup）的其他Reactor线程上，由其负责接入认证，握手等操作
    3. 应用层链路开始，将SocketChannel从主线程池对的Reator线程的多路复用器上摘除，重新注册到Sub线程池（workerGroup）的线程上，用于处理IO操作
    4. 负责HTTP/2服务端创建、HTTP/2请求消息的接入和响应发送

2. **gRPC service线程模型**

    1. gRPC服务端调度线程为SerializingExecutor，它实现了Executor和Runnable接口，通过外部的Executor对象，调度和处理Runnable，同时内部通过任务队列（ConcurrentLinkedQueue），通过run方法循环处理队列中存放的Runnable对象
    2. 负责gRPC的序列化和反序列化，以及应用服务接口的调用

3. **I/O通信线程模型**

    gRPC的做法是服务端监听线程和I/O线程分离的Reactor多线程模型

    工作流程如下：

    1. 业务线程发起创建服务端操作，在创建服务端时实例化两个EventLoopGroup。
    2. 服务端Selector轮询，监听客户端连接
    3. 若监听到客户端连接，则创建客户端SocketChannel连接，从workerGroup中随机选择一个NioEventLoop线程，将SocketChannel注册到该线程持有的Selecotr
    4. 通过调用EventLoopGroup的next()获取一个EventLoop(NioEventLoop)，用于处理网络I/O事件

    工作原理如下：

    ![](https://gitee.com/Euraxluo/images/raw/master/picgo/4a59ff78a4b550df92138c593e71771e.png)

4. **线程调度和切换策略**

    Netty Server I/O 线程的职责：

    1. gRPC 请求消息的读取、响应消息的发送
    2. HTTP/2 协议消息的编码和解码
    3. NettyServerHandler 的调度

    gRPC service 线程的职责：

    1. 将 gRPC 请求消息（PB 码流）反序列化为接口的请求参数对象
    2. 将接口响应对象序列化为 PB 码流
    3. gRPC 服务端接口实现类调用

    Netty Server使用的NIO线程是NioEventLoop，具有以下职责

    1. 作为服务端 Acceptor 线程，负责处理客户端的请求接入；
    2. 作为客户端 Connecor 线程，负责注册监听连接操作位，用于判断异步连接结果；
    3. 作为 I/O 线程，监听网络读操作位，负责从 SocketChannel 中读取报文；
    4. 作为 I/O 线程，负责向 SocketChannel 写入报文发送给对方，如果发生写半包，会自动注册监听写事件，用于后续继续发送半包数据，直到数据全部发送完成；
    5. 作为定时任务线程，可以执行定时任务，例如链路空闲检测和发送心跳消息等；
    6. 作为线程执行器可以执行普通的任务 Task（Runnable）。
    7. **NioEventLoop同时支持I/O操作和Runnable执行**：这样可以避免锁竞争，例如心跳检测，往往需要周期性的执行，如果 NioEventLoop 不支持定时任务执行，则用户需要自己创建一个类似 ScheduledExecutorService 的定时任务线程池或者定时任务线程，周期性的发送心跳，发送心跳需要网络操作，就要跟 I/O 线程所持有的资源进行交互，例如 Handler、ByteBuf、NioSocketChannel 等，这样就会产生锁竞争，需要考虑并发安全问题。

    1. 协议层消息的接收和编解码由Netty的I/O（NioEventLoop）线程负责
    2. 应用层的处理由应用线程gRPC相关线程负责，防止由于应用层耗时阻塞Netty的I/O线程

    线程切换，调度:

    1. 由于Netty线程和gRPC线程存在线程分工，因此需要频繁进行线程切换
    2. 这也是java版本gRPC的性能缺陷

    ![](https://gitee.com/Euraxluo/images/raw/master/picgo/b4ba273d77641bc440466ad7d37d70e9.png)

##### gRPC客户端线程模型

gRPC客户端线程由三类组成：业务调用线程，客户端连接和I/O线程。请求消息业务处理和响应回调线程，如下图所示：

![](https://gitee.com/Euraxluo/images/raw/master/picgo/77ffa44324ea0e318caa3693021ae490.png)



1. **应用线程**：负责调用gRPC服务端并获取响应，其中请求消息的序列化由该线程负责
2. **grpc-default-executor**线程池：客户端负载均衡以及Netty Client创建
3. **NioEventLoop 线程**：HTTP/2客户端链路创建，网络I/O数据的读写
4. **SerializingExecutor**：响应消息消息的反序列化
5. **responseFuture**:**SerializingExecutor**调用**responseFuture**的set(value),唤醒阻塞对的应用现线程，完成RPC调用

I/O通信线程模型

工作流程如下:

1. 由 grpc-default-executor 发起客户端连接，并且客户端只创建一个NioEventLoop，同时客户端使用EventLoop作为work线程
2. 发起连接操作，判断连接结果，判断连接结果，如果没有连接成功，则监听连接网络操作位SelectionKey。OP_CONNECT.如果连接成功，则调用pipeline().fireChannelActive()将监听位修改位READ
3. 由NioEventLoop的多路复用器轮询连接操作结果，判断连接结果，如果连接成功，重新设置监听为READ
4. 和服务端一样，由NioEventLoop线程负责I/O读写

工作原理如下所示：

![](https://gitee.com/Euraxluo/images/raw/master/picgo/f0b5c823ca2ee2a7ccc285ca8d081ffa.png)

##### 总结

1. **优点**：

    1. Netty线程模型

        Netty4之后，对线程模型进行了优化，通过串行化的设计避免线程竞争；并且减少了线程切换，避免额外的性能损耗

        Netty4采用串行化设计，将消息的读取，编码以及后续的Handler执行，都由I/O线程NioEventLoop负责，这样线程上下文就不用了进行切换。数据不会出现并发写的问题。

    2. gRPC线程模型

        消息的序列化和反序列化都由gRPC线程负责。因为Netty的I/O操作和业务Handler都是有NioEventLoop负责。但是某些CUP密集操作，适合放在业务应用的线程池中执行，否则会影响Handler串行执行操作。这样并发处理能力较均衡。

2. **改进点**:

    1. **将时间可控的接口调用直接在NettyI/O线程上处理**:

        gRPC采用的网络I/O线程和业务调用线程分离的策略，大部分场景下都最优。但是当接口逻辑简单，执行时间很短，不需要和外部网络，数据库进行交互，也不需要等待其他资源的。应该直接在Netty I/O线程中执行

    2. **减少锁竞争**：

        当前I/O线程和业务线程间没有任何的关联关系。也就是core*2个I/O线程和N个业务线程进行锁竞争

        可以通过线程绑定（通过一致性hash，将Netty的I/O线程和服务调用线程建立绑定关系），让锁竞争降低为1个I/O线程和N个业务线程进行绑定。提高性能

### gRPC服务调用原理

#### 服务调用方式

gRPC提供了多种服务调用方式

1. **同步服务调用**：最常用的服务调用方式，也是RPC/微服务默认的的调用方式

    工作原理：

    ​		客户端发起RPC调用，将请求消息路由到I/O线程，无论I/O线程是同步还是异步发送消息，发起调用的业务线程都会同步阻塞，等待服务端的应答，由I/O线程唤醒同步等待的业务线程，获取应答，然后业务流程继续执行。

    ​		同步服务调用会阻塞调用方的业务线程，为了防止服务端长时间不返回应答消息导致客户端用户线程被挂死，业务线程等待的时候需要设置超时时间，通常该值应该综合考虑业务端到端的通信延时，自身可靠性，超时时间不宜过大或者过小，在几百毫秒到几秒之间。

    1. 消费者调用服务端发布的接口，接口调用由服务框架包装成动态代理，发起远程服务调用

     	2. 通信框架的I/O线程通过网络将请求消息发送给服务端
     	3. 消费者业务线程调用通信框架的消息发送接口之后，直接或者间接调用wait()方法，同步阻塞等待应答
     	4. 服务端返回应答消息给消费者，由通信框架负责应答消息的反序列化
     	5. I/O线程获取到应答消息之后，根据消息上下文找到之前同步阻塞的业务线程。notify()阻塞的业务线程，返回应答给消费者，完成服务调用

    原理图如下：

    ![](https://gitee.com/Euraxluo/images/raw/master/picgo/47c58251d2f8bd98bfbcd72bcd14e421.png)

    

2. **并行服务调用**：对于无上下文依赖的多个服务，可以一次并行发起多个调用，这样可以有效降低服务调用的时延

    工作原理：

    ​		大多数业务应用中，服务总是串行化调用和执行，但是当多个服务之前没有上下文依赖关系，并且执行先后顺序没有严格要求，逻辑上可以被并行执行。又例如长流程业务，调用多个服务，对于时延比较敏感，其中有部分逻辑没有上下文关系，可以被并行调用和执行。

    1. 服务框架提供必将服务调用接口供接口消费者使用，一般形如`ParallelService.invoke(serviceName[],methodName[],args[])`
    2. 平台的并行服务调用器创建并行Future，缓存批量服务调用上下文信息
    3. 并行服务调用器循环调用普通的Invoker，通过循环的方式执行单个服务调用，获取到单个服务的Future之后设置到`Parallel Future`中；
    4. 返回`Parallel Future`给消费者
    5. 普通的Invoker调用通信框架的消息发送接口，发起远程服务调用
    6. 服务端返回应答，通信框架对报文做反序列化，转换成业务对象更新`Parallel Future`的结果列表
    7. 消费者调用`Parallel Future`的get(timeoout)方法，同步阻塞，等待所有结果全部返回；
    8. `Parallel Future` 通过对结果集进行判断，看所有的服务调用是否都已经完成（包括成功，失败，异常）
    9. 所有批量服务调用结果都已经返回，notify消费者线程，消费者获取到结果列表，完成批量服务调用，流程继续执行
    10. 通过批量服务调用和Future机制，可以实现并行服务调用，由于在调用过程中没有创建新的线程，用户就不需要担心依赖线程上下文的功能发生异常

    原理图如下：

    ![](https://gitee.com/Euraxluo/images/raw/master/picgo/c1585171726fbb13dfbf488b52ba34bb.png)

3. **异步服务调用**：客户端发起服务调用后，不同步等待响应，而是注册监听器或者回调函数，待接收到响应之后发起异步回调，驱动业务流程继续执行，比较常用的是Reactive响应式编程和JDK的Future-Listener回调

    工作原理：

    ​		JDK原生的Future只要用于异步操作，它代表了异步操作的执行结果，用户可以通过调用它的get方法获取结果。如果当前操作没有执行完，get操作将阻塞调用线程。实际项目中，往往会扩展JDK的Future，提供Future-Listener机制，支持主动获取和被动异步回调通知两种模式，适用于不同的业务场景。

    ​		异步服务调用的优点：提交服务调用效率，减少业务线程阻塞啥时间，避免业务线程阻塞

    1. 消费者调用服务端发布的接口，接口调用由服务框架包装为动态代理，发起远程服务的调用
    2. 通信框架异步发送请求消息，如果没有发生I/O异常，返回；
    3. 请求消息发送成功后，I/O线程构造Future对象，设置到RPC上下文中
    4. 业务线程通过RPC上下文获取Future对象；
    5. 构造Listener对象，将其添加到Future中，用于服务端应答异步回调通知。
    6. 业务线程返回，不阻塞等待应答
    7. 服务端返回应答消息，通信框架负责反序列化工作
    8. I/O线程将应答设置到Future对象的操作结果中
    9. Future对象扫描注册的监听器列表，循环调用监听器额的operationComplete方法，将结果通知给监听器，监听器获取到结果后，继续后续业务逻辑的执行，异步服务调用结束

#### 服务调用的误区

1. I/O异步服务就是异步

    通信框架基于NIO实现，并不意味着服务框架就支持异步服务调用了

    在RPC/微服务框架中，引入NIO的好处有：

    - 所有的I/O操作都是非阻塞的，避免有限的I/O线程因为网络，对方处理慢等原因被阻塞
    - 多路复用的RReactor线程模型，基于Linux的epoll和Selector，一个I/O线程可以并行处理成百上千链路，解决了传统同步I/O通信线程膨胀的问题

    NIO只解决了通信层面的异步问题，和服务调用的异步没有必然关系。

    **通信框架和异步服务框架的关系**：

    ​		用户发起远程服务调用之后，经历层层业务逻辑处理，消息编码，最终序列化后的消息会被放入到通信框架的消息队列中。业务线程可以选择同步等待，也可以选择直接返回。通过消息队列对的方式实现业务层和通信层的分离是比较典型的做法。

    ​		通信层采用NIO或者是BIO对上层的业务是不可见的，双方的汇聚点就是消息队列，在JAVA中通常为Queue，业务线程将消息放入到发送队列中，可以选择主动等待或者立即返回，和通信框架是否是NIO的没有任何关系。因此不能认为I/O异步就代表服务调用也是异步。

    如图所示：

    ![](https://gitee.com/Euraxluo/images/raw/master/picgo/a69619ff7aad67e7a714925c69642eb8.png)

2. 异步服务调用性能更高：

    - 对于I/O密集型，资源不是瓶颈，大部分时间都在同步等待应答，异步服务调用可以带来巨大的吞吐量提升，资源使用率也可以提高，更加充分的利用硬件资源提升性能。

    - 对于时延不稳定的接口，如第三方服务的响应速度，数据库操作，异步服务也会带来性能提升
    - 但是如果接口调用的时延本身就小(毫秒级)，内存计算型，不依赖第三方服务，内部也没有I/O操作，异步服务调用并不会提升性能

3. Restful接口的问题和收益

    **问题**：

    - 潜在性能风险：若RestfulAPI底层使用的HTTP协议栈是同步阻塞I/O，则服务端的处理性能将大打折扣

    **收益**：

    - 接口更加规范和标准，可以通过Swagger API规范来描述服务接口，并生成客户端和服务端代码

    - Restful API可读性更好，更容易维护
    - 服务提供者和消费者基于API契约，双方可以解耦，不需要在客户端引入SDK和类库的直接依赖，未来的独立升级也更加方便
    - 内外使用同一套API，非常容易开放给外部或者合作伙伴使用

    **解决方案**：

    - 如果选择RestfulAPI作为内部RPC或者微服务的接口协议，则建议使用HTTP/2.0 协议，优点：支持双向流，消息头压缩，单TCP的多路复用，服务端推送等。效果和基于TCP的私有协议类似。

#### gRPC服务调用

gRPC的通信协议基于标准的HTTP/2设计，主要提供了两种RPC调用方式

1. 普通的RPC调用，请求-响应模式
2. 基于HTTP/2.0 的streaming调用方式

##### 普通RPC调用

1. 同步阻塞式服务调用

    实现类是xxxBlockingStub，gRPC框架的ClientCalls在框架层做了封装，当异步发起服务调用后，会同步阻塞调用方线程。直到收到响应(queue.task()方法)再唤醒被阻塞的业务线程

2. 异步非阻塞调用

    基于Future-Listener机制，通常实现类是xxxFutureStub。当调用这种实现类的方法时，返回的不是应答，而是ListenableFuture，将ListenableFuture加入到gRPC的Future列表中，创建一个新的FutureCallback对象，当ListenableFuture获取到响应之后，gRPC的DirectExecutor线程池会调用新创建的FutureCallback，执行onSuccess或者onFailure，实现异步回调通知

3. 异步非阻塞调用

    基于Reactive的响应式编程模式，通常实现类是xxxStub。框架会构造响应StreamObserver，通过响应式编程，处理正常和异常回调，将响应StreamObserver作为入参传递到异步服务调用中后，该方法返回空，程序继续执行不会产生阻塞。当收到响应消息时，调用StreamObserver的onNext方法。当Straming关闭时，调用onCompleted方法。Reactive风格的异步调用，相比于Future模式，没有任何同步阻塞点，无论是业务线程还是gRPC框架的线程都不会同步等待。

##### Streaming模式服务调用

1. 服务端streaming方式

    ` rpc ListFeatures(Rectangle) returns (stream Feature) {}`

    服务端streaming模式，指的是客户端一个请求，服务端返回N个响应，每个响应可以返回。服务端Streaming模式也支持同步阻塞，和Reactive异步两种调用方式。

2. 客户端streaming

    `rpc RecordRoute(stream Point) returns (RouteSummary) {}`

    与客户端发送多个请求，服务端返回一个响应，多用于聚合计算。

    异步服务端调用获取请求StreamObserver对象，循环调用requestObserver.onNext(point),异步发送请求消息到服务端，发送完成后，调用requestObserver.onCompleted(),通知服务端所有请求已经发送。

3. 双向streaming

    `rpc RouteChat(stream RouteNote) returns (stream RouteNote) {}`

    客户端发送N个请求，服务端返回N个或者M个响应。该特性可以充分利用HTTP/2.0的的多路复用功能。HTTP/2.0的链路上，请求和响应可以同时存在，实现全双工通信。该方式，只支持异步通信。

### gRPC安全性设计

##### 敏感数据加密传输

1. 基于SSL/TLS的通道加密

    当存在跨网络边界RPC调用时，往往会需要通过TLS/SSL对传输通道加密，以防止请求和响应消息中的敏感数据泄露。跨网络边界包括:

    - 后端微服务开放给端侧，例如手机等，没有统一的API Gateway/SLB做安全接入和认证
    - 后端微服务直接开放给DMZ部署的管理或者运维类Portal
    - 后端微服务直接开放给第三方合作伙伴/渠道

    场景如下图所示：

    ![](https://gitee.com/Euraxluo/images/raw/master/picgo/1a69fa3b2939b510dc9af185e66e3d48.png)

1. 针对敏感数据的单独加密

    当RPC调用并不涉及敏感字段或者，敏感字段的占比较低，为了最大程度的提升吞吐量，降低调用时延，通常会采用HTTP/TCP+敏感字段单独加密的方式，即保障了敏感信息的传输安全，同时也降低了采用SSL/TLS加密通道的性能损耗。

    业务方，通常有Handler拦截，对请求和响应消息进行统一拦截，根据注解或者加解密标识对敏感字段进行加解密，避免侵入业务。

    原理如下：

    ![](https://gitee.com/Euraxluo/images/raw/master/picgo/6507b91b6cae0bbcaea4ec535d5944dd.png)

    缺点：

    - 对敏感信息的识别可能存在偏差，容易遗漏或者过度保护，需要解读数据和隐私保护方面的法律法规，而且不同国家对敏感数据的定义也不同，这会为识别带来很多困难
    - 接口升级时容易遗漏，例如开发新增字段，忘记识别是否为敏感数据。

##### gRPC安全机制

1. 通道凭证：默认提供了基于 HTTP/2 的 TLS，对客户端和服务端交换的所有数据进行加密传输
2. 调用凭证：被附加在每次 RPC 调用上，通过 Credentials 将认证信息附加到消息头中，由服务端做授权认证；
3. 组合凭证：将一个频道凭证和一个调用凭证关联起来创建一个新的频道凭证，在这个频道上的每次调用会发送组合的调用凭证来作为授权数据，最典型的场景就是使用 HTTP S 来传输 Access Token；
4. Google 的 OAuth 2.0：gRPC 内置的谷歌的 OAuth 2.0 认证机制，通过 gRPC 访问 Google API 时，使用 Service Accounts 密钥作为凭证获取授权令牌。

##### SSL/TLS工作原理

SSL/TLS 分为单向认证和双向认证，在实际业务中，单向认证使用较多，即客户端认证服务端，服务端不认证客户端。

单向认证原理如下：

1. SL 客户端向服务端传送客户端 SSL 协议的版本号、支持的加密算法种类、产生的随机数，以及其它可选信息；
2. 服务端返回握手应答，向客户端传送确认 SSL 协议的版本号、加密算法的种类、随机数以及其它相关信息；
3. 服务端向客户端发送自己的公钥；
4. 客户端对服务端的证书进行认证，服务端的合法性校验包括：证书是否过期、发行服务器证书的 CA 是否可靠、发行者证书的公钥能否正确解开服务器证书的“发行者的数字签名”、服务器证书上的域名是否和服务器的实际域名相匹配等；
5. 客户端随机产生一个用于后面通讯的“对称密码”，然后用服务端的公钥对其加密，将加密后的“预主密码”传给服务端；
6. 服务端将用自己的私钥解开加密的“预主密码”，然后执行一系列步骤来产生主密码；
7. 客户端向服务端发出信息，指明后面的数据通讯将使用主密码为对称密钥，同时通知服务器客户端的握手过程结束；
8. 服务端向客户端发出信息，指明后面的数据通讯将使用主密码为对称密钥，同时通知客户端服务器端的握手过程结束；
9. SSL 的握手部分结束，SSL 安全通道建立，客户端和服务端开始使用相同的对称密钥对数据进行加密，然后通过 Socket 进行传输

流程如下图所示：

![](https://gitee.com/Euraxluo/images/raw/master/picgo/672ce5cc60be10d880553ff883c953ee.png)

### gRPC序列化机制

常用的序列化机制：json序列化，MessagePack序列化，Thrift序列化框架，Protocol Buffers 序列化框架

#### Thrift 序列化框架

thrift是facebook开源的，支持多语言的高性能通信中间件。他能提供序列化和多种的类型的RPC服务

Thrift由5部分组成

1.**语言系统以及IDL编译器**：负责将IDL文件生成对应语言的接口代码

2.**TProtocol**：RPC协议层，可以选择多种不同的序列化方式，如果JSON，Binary

3.**TTransport**:RPC传输层，可以选择不同的传输层实现：如socket，NIO，MemeoryBuffer等

4.**TProcessor**:作为协议层和用户提供的服务实现之间的纽带，负责调用服务实现的接口

5.**TServer**：聚合Tprotocol，TTransport和TProcessor等对象

其中编解码框架是Tprotocol部分。

#### TProtocol codec:

与Protocol Buffers 类似，Thrift通过IDL描述接口和数据结构定义，它支持8种Java基本类型，Map，Set，List 支持可选和必选定义。

Thrift支持三种编解码方式：

1. 通用的二进制编解码
2. 压缩二进制编解码
3. 优化的可选择字段压缩编解码



#### MessagePack 序列化框架

MessagePack是一个高效的二进制序列化框架，可以向json一样支持不同语言间的数据交换，但是性能更好，序列化后的码流也更小

MessagePack支持多语言，并且API设计也像Json序列化一样简单



#### Protocol Buffers序列化框架

Protocol Buffers 是一个可以独立使用的序列化框架，它并不与grpc框架绑定，任何需要支持多语言的RPC框架都可以选择使用Protocol Buffers作为序列化框架

Protocol Buffers 的使用包括：

- IDL文件定义（*.proto）,包含数据结构定义，以及可选的服务接口定义（gRPC）；
- 各种语言的代码生成（含数据结构定义，序列化和反序列化接口）
- 使用Protocol Buffers的API进行序列化和反序列化

**Netty 中使用**：

Netty 提供了对于Protocol Buffers 的支持，在服务端和客户端创建时，只需要将Protocol Buffers 相关的CodeC Handler加入到ChannelPipeline中即可

#### gRPC序列化

gRPC默认使用Protocol Buffers作为RPC序列化框架，通过Protocol Buffers 对消息进行序列化和反序列化，然后通过Netty的HTTP/2，以Stream 的方式进行数据传输。

##### 1.客户端请求消息序列化

客户端通过build模式构造请求消息，然后通过同步/异步方式发起RPC调用，gRPC框架负责客户端请求消息的序列化，以及HTTP/2 Header和Body的构造，然后通过Netty提供的HTTP/2协议栈，将HTTP/2请求消息发送给服务端

##### 客户端请求消息发送流程

1. **请求消息的构建**：使用Protocol Buffers生成的代码，通过build模式对请求消息设置，完成请求消息的初始化

2. **请求消息的序列化**：使用Protocol Buffers 的Marshaller 工具类，对于生成的请求对象进行序列化，生成ProtoInputStream

3. **请求消息的首次封装**：主要用于创建NettyClientStream，构造gRPC的HTTP/2消息头等

4. **请求消息的二次封装**：将序列化之后的请求消息封装成SendGrpcFrameCommand，通过异步的方式由Netty的NIO线程执行消息的发送

5. **NettyClientHandler处理**：gRPC的NettyClientHandler拦截到write请求后，根据Command类型判断是业务消息发送，调用Netty的Http2ConnectionEncoder，由Netty的HTTP/2协议栈创建HTTP/2 Stream并最终发送给服务端

6. **线程模型**：请求消息构建，请求消息序列化，请求消息封装都由客户端用户线程执行；请求消息的发送由Netty的NIO线程执行。

    发送流程如下图所示：

![](https://gitee.com/Euraxluo/images/raw/master/picgo/79497e27b54667022f918ca1c05df004.png)

##### 2.服务端请求消息反序列化

服务端接收到客户端的HTTP/2请求消息之后，由Netty HTTP/2协议栈的FrameListener.onDataRead方法调用gRPC的NettyServerHandler，对请求消息进行解析和处理。

##### 服务端读取客户端消息反序列化流程

1. **Http/2内容读取**：通过FrameListener监听onDataRead，获取HTTP/2消息内容

2. **构造NettyServerStream**：通过HTTP/2 Stream构造gRPC NettyServerStream，，并将内容拷贝到NettyReadableBuffer中

3. **解析Body**：调用MessaeDeframer解析请求消息体。并且由gRPC的SerializingExecutor负责body的解析

4. **反序列化请求消息**：使用Marshaller对请求消息做反序列化。

5. **线程模型**：Netty HTTP/2消息的读取和校验，由Netty NIO线程负责。后续HTTP Body 的反序列化，则由gRPC的SerializingExecutor 线程池完成

    数据流图如下所示：

    ![](https://gitee.com/Euraxluo/images/raw/master/picgo/136c6ceb74178b4205c305373f376951.png)

##### 3.服务端响应消息序列化

服务端接口调用完成之后，需要将响应消息序列化，然后通过 HTTP/2 Stream（与请求相同的 Stream ID）发送给客户端。

##### 服务端响应消息流程

1. 服务端的接口实现类中调用responseObserver.onNext(reply),触发响应消息的发送流程

2. 响应消息的序列化：使用Protocol Buffers的Marshaller工具类，对于生成的响应对象进行序列化，生成ProtoInputStream

3. 对HTTP响应Header进行处理，包括设置响应消息的content-length 字段，根据是否压缩标识对响应消息进行gzip压缩等

4. 对响应消息进行二次封装，将序列化之后的响应消息封装成SendGrpcFrameCommand，通过异步的方式由Netty的NIO线程发送

5. gRPC的NettyServerHandler拦截到write的请求消息之后，根据Command类型判断是业务消息发送，调用Netty的Http2ConnctionEncoder，由Netty的HTTP/2协议栈创建Http/2 Stream 并最终发送给客户端

6. **线程模型**：响应消息的序列化以及HTTP Header的初始化等操作由gRPC的SerializiingExecutor线程池负责。HTTP/2消息的编码以及后续发送，由Netty的NIO线程池负责

    数据流图如下所示：

![](https://gitee.com/Euraxluo/images/raw/master/picgo/ce7b17e8b52d93e3923c8007e0703b0e.png)

##### 4.客户端响应消息反序列化

客户端接收到服务端响应之后，将 HTTP/2 Body 反序列化为原始的响应消息，然后回调到客户端监听器，驱动业务获取响应并继续执行。

##### 客户端响应消息反序列化流程

1. 读取消息：与服务端类似，通过Netty HTTP/2 协议栈的FrameListener监听并回调gRPC Handler（NettyClientHandler）,读取消息。

2. 获取stream对象： 根据streamId，获取Http2Stream，通过Http2Stream的getProperty方法获取NettyClientStream

3. 解析响应消息体： 调用MessageDeframer的deframe 方法，对响应消息体进行解析。客户端和服务端实现机制不同（通过不同的Lisstener重载messageRead方法）

4. 线程切换：调用ClientStreamListenerImpl的messageRead进行线程切换，将反序列化操作切换到gRPC工作线程或者客户端业务线程中（同步阻塞调用）

5. 调用Protocol Buffers的Marshaller对响应消息进行反序列化，还原成原始对的message对象

    数据流图如下所示：

    ![](https://gitee.com/Euraxluo/images/raw/master/picgo/587956281615406747c0d4ac959b450b.png)