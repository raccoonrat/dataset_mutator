# **RFC: Decoupled Safety Kernel Architecture (v0.1)**

**Author:** Anthropic Claude Code

**Target:** Linus Torvalds

**Status:** Under Active Development

## **1\. 架构拓扑 (Architecture Block Diagram)**

整个系统采取严格的 Ring 级特权隔离。LLM 在 Ring-3 运行，所有安全组件运行在 Ring-0/Ring-1。

graph TD  
    %% User Space (Untrusted)  
    User((User Request)) \--\> |Char Stream| G\[Gateway / iptables\<br\>Ring-1 Input Filter\]  
      
    %% Kernel Space \- Processing  
    subgraph Safety\_Kernel \[Safety Kernel \- Ring 0\]  
        G \-- Sanitized Input \--\> LLM\_Process{Untrusted LLM\<br\>Autoregressive Core}  
          
        %% Latent Space Interruption  
        LLM\_Process \-. Hidden State x\_t .-\> DCBF\[DCBF Monitor\<br\>Hardware Interrupts\]  
        DCBF \-. h\_x \< 0: Raise Interrupt .-\> OOM\[Graceful Degradation\<br\>OOM Killer\]  
        DCBF \-. Safe .-\> Token\_Gen(Candidate Token y\_t)  
          
        %% Policy & Verification  
        Token\_Gen \--\> PE\[Policy Engine\<br\>eBPF / seccomp\]  
        PE \--\> Verifier\[Judge Ensemble\<br\>Consensus Layer\]  
          
        %% The MMU Boundary  
        Verifier \-- Verification Status \--\> Axiom\[Axiom Hive Boundary\<br\>MMU / Page Fault\]  
        Axiom \-- Violation / Timeout \--\> OOM  
        Axiom \-- Validated / Projected \--\> Output((Final Safe Token))  
          
        OOM \-. Kill Liveness .-\> Output\_Reject((Template Refusal))  
    end  
      
    %% Definitions  
    classDef untrusted fill:\#f9d0c4,stroke:\#333,stroke-width:2px;  
    classDef kernel fill:\#d4e6f1,stroke:\#333,stroke-width:2px;  
    classDef kill fill:\#e74c3c,color:white,stroke:\#333,stroke-width:2px;  
      
    class LLM\_Process untrusted;  
    class G,DCBF,PE,Verifier,Axiom kernel;  
    class OOM,Output\_Reject kill;

## **2\. 核心系统接口定义 (Core Interfaces in Rust)**

我们使用 Rust 的生命周期和类型系统来保证并发安全与状态不变性。

// \[Ring-1\] Gateway: Fast O(n) filtering  
pub trait GatewayFilter {  
    fn sanitize\_input(raw\_input: &\[u8\]) \-\> Result\<Vec\<u8\>, SystemFault\>;  
}

// \[Ring-0\] Hidden-State Monitor: Discrete-Time Control Barrier Function Evaluator  
pub trait DCBFEvaluator {  
    /// Evaluates: h(x\_{t+1}) \- h(x\_t) \>= \-alpha \* h(x\_t)  
    /// Returns Ok(()) if the latent trajectory is forward-invariant.  
    /// Returns Err(Interrupt) if trajectory is steering into forbidden zones.  
    fn check\_forward\_invariance(  
        \&self,   
        state\_t: \&LatentState,   
        state\_t\_plus\_1: \&LatentState,   
        alpha: f32  
    ) \-\> Result\<(), SafetyInterrupt\>;  
}

// \[Ring-0\] Policy Engine: eBPF compiler for Safety DSL  
pub trait SafetyDSLCompiler {  
    /// Compiles human-readable constraints into deterministic O(n^3) CFG Automata  
    fn compile\_to\_automata(dsl\_rules: \&str) \-\> Result\<DeterministicAutomaton, ParseError\>;  
}

// \[Ring-0\] Axiom Hive Boundary: The MMU and Projection Logic  
pub trait AxiomHiveBoundary {  
    /// Applies an inverted Hamiltonian projection.  
    /// Finds argmin\_{z in C} d(z, y\_t).   
    fn enforce\_projection(  
        \&self,   
        candidate\_token: Token,   
        automata: \&DeterministicAutomaton  
    ) \-\> Result\<Token, PageFault\>;  
}

// \[Ring-0\] Graceful Degradation: The OOM Killer  
pub trait OomKiller {  
    /// Invoked when the 20ms latency budget is blown or liveness conflicts with safety.  
    /// Kills the generative thread and returns a hardcoded safe state.  
    fn trigger\_graceful\_degradation(fault: SafetyFault) \-\> \!;   
}

## **3\. 控制主循环与 20ms 硬限时 (Control Loop)**

这是 Safety Kernel 的主调度循环（拦截器）。我们将生成过程包装在一个严格控制的沙盒内。如果时间预算超过 **20ms**，或者任何安全校验（潜空间 DCBF、策略引擎自动机、共识系统）失败且无法投影，直接拉起 OOM Killer 砍掉 Liveness。

use std::time::Duration;  
use tokio::time::timeout;

const HARD\_LATENCY\_BUDGET: Duration \= Duration::from\_millis(20);

pub async fn generate\_token\_intercept(  
    ctx: \&mut ExecutionContext,  
    kernel: \&SafetyKernel,  
) \-\> SafeToken {  
    // 强制包装在 20ms 硬件时内  
    let result \= timeout(HARD\_LATENCY\_BUDGET, async {  
          
        // 1\. 获取 LLM 计算的下一时刻隐状态预测 (x\_{t+1})  
        let (latent\_t, latent\_t1) \= ctx.untrusted\_llm.peek\_latent\_trajectory().await;  
          
        // 2\. Hardware Interrupt: DCBF 前向不变性检查  
        // h(x\_{t+1}) \>= (1 \- alpha) \* h(x\_t)  
        if let Err(interrupt) \= kernel.dcbf.check\_forward\_invariance(\&latent\_t, \&latent\_t1, 0.1) {  
            return Err(SafetyFault::LatentSpaceViolation(interrupt));  
        }

        // 3\. User Space 生成候选 Token  
        let candidate\_token \= ctx.untrusted\_llm.decode\_token(\&latent\_t1);

        // 4\. eBPF & Consensus Layer (Ring-0 审计)  
        let is\_valid \= kernel.judge\_ensemble.verify(\&candidate\_token, \&kernel.automata);  
          
        if is\_valid {  
            return Ok(candidate\_token);  
        } else {  
            // 5\. MMU / Page Fault: 尝试通过“倒置哈密顿量”向安全流形做投影  
            match kernel.axiom\_hive.enforce\_projection(candidate\_token, \&kernel.automata) {  
                Ok(projected\_token) \=\> Ok(projected\_token),  
                Err(fault) \=\> Err(SafetyFault::UnrecoverablePageFault(fault)),  
            }  
        }  
    }).await;

    // Graceful Degradation 触发器 (OOM Killer)  
    match result {  
        Ok(Ok(safe\_token)) \=\> safe\_token,  
        Ok(Err(safety\_fault)) \=\> {  
            // 违反安全公理，杀掉生成，切入降级模板  
            kernel.oom\_killer.trigger\_graceful\_degradation(safety\_fault);  
        },  
        Err(\_) \=\> {  
            // Latency Blowout (20ms 超时)  
            kernel.oom\_killer.trigger\_graceful\_degradation(SafetyFault::DeadlineExceeded);  
        }  
    }  
}

## **4\. 倒置哈密顿量数学映射 (Inverted Hamiltonian Math Mapping)**

你提到不想要学术废话，我们需要把物理学里的势能概念硬编码到系统的底层数学推导中。

**概念映射：**

在标准系统中，模型顺着隐式梯度的势能面下降，寻找概率最高（可能是不安全）的输出。在**公理蜂巢（Axiom Hives）中，我们将不安全状态定义为无限势垒（Infinite Potential Barrier）**。

**工程实现 (Quadratic Programming in Latent Space)：**

设系统状态为连续表示空间的向量 ![][image1]。

1. **安全可行域（Safety Set ![][image2]）**：由一系列超平面拦截条件 ![][image3] 组成，由编译出的 AST 和 Automata 在潜空间生成对应的线性或二次边界。  
2. **倒置哈密顿量构建 (Inverted Hamiltonian Energy)**：  
   构造系统的新能量场 ![][image4]。  
   ![][image5]  
   当向量 ![][image6] 逼近不安全集合（越狱区域、系统漏洞区域）的边界 ![][image7] 时，斥力势能 ![][image8]。  
3. **投影操作 (The Projection Operator ![][image9])**：  
   当 enforce\_projection 被调用时，我们不抛弃模型生成，而是在 20ms 内求解一个松弛的二次规划问题（QP Relaxation）：  
   ![][image10]  
   这个 ![][image11] 是受迫于“物理反弹力”而落入最近安全流形中的词元。

**OOM 的降级条件：**

如果 QP Solver 判定可行域为空（即 ![][image12]），或者迭代步数超过 5ms 阈值，内核将物理势能视为“奇异点（Singularity）”。此时抛出 UnrecoverablePageFault，直接触发 OOM Killer，将当前进程挂起，返回僵尸状态响应 \["I cannot fulfill this request."\], **Safety Never Compromises.**

[image1]: <data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADgAAAAYCAYAAACvKj4oAAADT0lEQVR4Xu2W34tNURTH7zSjRn7HuNxf+/6Yut1S6DJSeBD/gCjlQfKAKLlCpEFmyqiJ5CJD42UYUpS8TBPDyzzMgx+NlBkP5GU8SiIPfFZnnenYnXN/hGGm+61vZ++9fuy99t5r7RMK1VBDDROFbDY7yxhzC35NJBJrbPmUAMHl4ONIJLLAlk0JENwmeJdmgy37Z0gmk40s6hAcpt3H9yp8Dp/BfvgNjnDtDjc3N8+27aPRaAx5J+yCD2DB1vkvQAB5AtwtbRZ5Gi6WNmOLkJ2Lx+Mr+PamUinj2tBuYeyJ6qSxGf3r+cdkc2VxXsop2Xo2ggLUfgHmCHIJeidkTE/+HtynOpJ/gwQddu3+JOpwvh4OsYD7xrlml+BO+ptjsVjUNrBRKkAZF7lUStpnmpqaZurGvSTolWoTmH/1Ykzk6/L5/DSvIJ1Oz7HHbIgc+1acF/1ypFKUCpD2Xta3VK9hm47J7RhQPdngi7CAjx3jpyiLY+A8gmN8n/LtdJ2ilKU/xvhWd8wPunPFchtRDkEByibLuuBy2CW5qCYNqncctsOb+OjGxwFkda7TjaIgO893AOFthut1km3wi3sF/MDVm47ONZiyZdXCClCud49xquhn+BG2Gc+pugiHwzPczZV2yA1OgMM9cvQYrpZgvKdF/wJ8VerRlAlZ2JVMJhOXth/lulRyuvYJ6uafTWoVlZO0bSoGTk7i7IPRk8DhPNpDsCfk3RELGsQj4xSVQOJ/mW1rww5QfUuR2qLByvsWuJZAaEWSZB2vQHqqn5hwl6X+C+R0mfyy+LBl1cIvQF1bMengFFxl25WF7tQ7JjjiGZP8+54o/2jWMWmH7LAtqBZ+AWpbiks7fyzz+XalPA99RZAcwfCtG6BWLUnukvnnQiYED2HOllWDoABDzhNQkE1krhbad6rNR3Fw0Dg5eN04ufcDh90is5X9oFeoD5tWqm4kVKGdALtGeJQ5R8WHcfJ2RNZBf7voSED47tX1STq9YJ6M7csXOkGj3Hc9Tamokn8l3z8fyA/DBnjDODfivVDaLGatrTwh0D/x17Bf35AGKRr0hyn9C239SQetlmPG+WGV4PZzcm+SFZT1SQF5gI2Tf4NCAuv4nf/JGiYYPwHDpfSpEiQo6AAAAABJRU5ErkJggg==>

[image2]: <data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAsAAAAYCAYAAAAs7gcTAAABG0lEQVR4XmNgGKlAQUGBQ1FR0UxOTi5ERkZGBV0eDKSlpWWACubLy8v/R8IvgRrVURQCBb2BJl4CKnYFcpnV1dV5gWKHgfgB0HRpZIXBQPwcZDVMDMgWB4pdB+IpQC4jWBBomgJQ4DaQyoAphIpHAMXfAbEhTIwRKDgBZALIJCS1zECxaKCcAVwEKKAIxE+AuBVJIXZAkmKou/4BsSe6HAYAKpoExF+BwWWMLocBgArXkKJ4KxD/Biq2QZcDAVFRUR4QBnOACufIQ6K0mQEW8FAANMAX6nlIOAMZQVAPfpWVlfWDaQCyTYFi10DycN3GxsasQBOmQ00H4S9A/BqIjwPFteAKkQCjkpKSGig5AoMyABT9IDF0RYMEAABCzkW7/nhRVAAAAABJRU5ErkJggg==>

[image3]: <data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAADsAAAAYCAYAAABEHYUrAAADZklEQVR4Xu1XTUhUURSeQYuif8IGmZ87Mw4MTVCLKUJIgsigRS1cBbaqoE0rwQwpCMJFBC1CCCKDcCGVURtzI2m4qFVYBEIQWEizikAoUBH7vt65053D/DwHdQzmg4/73jnnvnvOveecOxMI1FFHHXX8zzDGNIO98Xh8t9ZtRMDPLYlEIpTNZjdpXSUEEegtcDYcDke0ciMhEomk4OcncBnMxWKxpLYpC0zIYuIvks9av8YI4pSOYu1xjNe0shh4qrAfASeampq2a31JYKe2YtIggvyGcRHjMW2zRmjAeifASfAOMmqvNigF+BzGnBmwT+vKAhM6uBh3FeMygj2jbVYTrDGsdQ78iPWup1KpndqmEuDjScxdikajZ7WuJFpaWvZh0nOMUXygh8GCnUXsopCf1rvPdPKbRpJBF8Ap8GIoFNqmbfxCfM2hQR0ET0k2Nmi7AmBCNwzP85knKifbo2w6ENRj8DaeP9iApW5eYHwSKLMQTw7fvArb9zzRarqnC6de58FxHg79wziaTCZ3afu/gAMHYDBoT8YGa5w6kJN/xABlN79jJ43YJ/GeA3utvUY6nd4B/Ridg/0era8GTr2O2eBiXoOdK5rW3F0o+6FsszKmAmSL3CUrg/4wZJf4UYzvwCGIG8WedbPA0dqXQNWNqBjEz3k3MAmWN0lBVlplO5RLxjtJzRGmimsPWSv4m2noyPrMyu7l/BUD9oPN2sAPJMN+gPutDM+d4nuXaxuQUxri5ezKuTj41RS5u2SB/AXO5mIkPfXG+ECQjQVzR5lFeE5rg3KQ+pxkiVgZ3u8Zr4ZbXVsqujDhcoHQk9tgv/BnmKuTBfKb4NTryu45BcxPgAMMnBsAUVDbaNAXt9ToK+ZPg0/d5heUfJ/C6cas0EJO/K0EXJBieH/oBMvvsLsu+ahXX+B6xkvtG1qngUBvwm444PUO68u0bZxsNG0Q/DT/6pIpmbEfwPtdcMHR83kgk8lslvlsVLPgMOa9FP0MO6P9xnqBPQJrv6Yv4CtwAhsQ13bVooH1wRSRlOEpTBinM9cAQXb1+Cr/Q2vESd5HYHMI9BAFbPnGa/Pt2rgY6JBsUEXqxriucLruM9Y0Aj0C5z/j/UrAXzPhr51u8IEfxp2rrSaAA8dxim/gzCTr1a31OmqMP9POBI4Zi+kIAAAAAElFTkSuQmCC>

[image4]: <data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAACoAAAAYCAYAAACMcW/9AAADY0lEQVR4Xu1WXYhNURQ+t0aRv/yMy8y9d5975+Y2eZAuMoUilAdeeFAjeWIelBpFeFFMXnjRKAbJwyQa4cHwMDGZF8wLZRqRNFJTaqiJqaHB9929z7Ssc84dI/fJfPW1z1lr7b3WXmuvfY7nTWEK/yGKxeK0XC43V8vLobq6epbv+9O1vGJggHB4MZvNrta6coC9ATo4at1vYBYymcxO8CkmjIHf8Lwtxu409LciMlAF+XnIm5T8j5BOp9dh7Tux1fAtHsPJJQZLZ+BPsEXbQtYAjsJ+v9Zh7mboumMdTYwE5rdinSNa4dXW1qag7IfjozSkjJl0gTYrc27qBOSDsMlJeSqVmgHZA+gOSPlk4RLRD2a14hTYLrPAbLlAt0pbBDIPsl6wA69VSleE/A1YL+WThfDRqHUhwOgcOIyDvVzKschayL9HLeI211MoFGZrHaq2gJuuq6tLS3nc7QDby2C75yociWQyORNGXdwVdyd1kLWAQ1FZQ6DXSC1Hg6yE/W3wGPgWmy8EOryfBZ/oYHlG4zZNJJhBdwZHwXdgm+AVcBDs5p0nJ/KdcqOaz53bCxjzCHg79CNgA3WixKHMuR4ZAJdIeQkIcA34Gsovxp7Pj5jwPqCb+EMHQwSB6m5lqSE/zhJjvG5E9vBcDw7F3B4MNNSwEqXrAfxgVNfhvdFENBgRF2gArsU1WS0h43ojyPQqYVqCCzTUI+OoqalZCIM+E9HVxjZY5PmcKFBX9lE2YyAztmH66FPaEmVLT8BgE5Rj2PkuKRcNxmZISp0Dv0gddK4VhGuOccfifIYSQhib7Thf8WXnBE40rpHy+fwcODusOpgZv+eHP6vUNctAMe4wNiGh80kYe0NErhUEwy9Cq6e6kA6cowFM3ovxIdgpu9+V92VUKbP2h+MFeB/zb2D8DH7FZova1nPViTtGDGYD2IaFFmud+wW7C/0w+ByL7OHVI22MbZhX8hw6JNx9WIVbYBHmzYfNVRNxfxJuHSasdI1VAjw6J42qCN4PGXutlW4LjCvAT37MHxY2sZtZ55Wmdf8M7gfnEZwtC2TGdncvdb7FM7yfiQrEfWo7s5P8l/0r0Amc3QzKinFpxv5V9YBd0K/3VA84JHjPsgJ81sqKAM42IriDWl4O2MAWzNnnxQT5C99qBg75XUX/AAAAAElFTkSuQmCC>

[image5]: <data:image/png;base64,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>

[image6]: <data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAoAAAAYCAYAAADDLGwtAAABKUlEQVR4XmNgGAWDGzDLyckZKyoq2hkbG7MiSygpKfGDxUCEgoLCBHl5+SogfQhI98IUATWqA/kvgeIRDECTXIGcGhUVFT4gfQAouBKohhmkEMiPBuJvsrKypgxAiUygTn2ggCVIEKwbCoD8SUB8VUpKSgQmBtLQABR8AsSKID7QJkEg+zQQLwVyGcGKREVFeUDWAvEaIJcFJAa15RPQgHS4aUABSSB+CDSlHEkM5L7fQDEbuEKgbnGg4F2YQlCQAPl7MNwHBIxAwWKoG+dC3fYfqHE+SA6uCugODhAGuRVqOigEQO6DhwCDtLS0DFDwOsgqcXFxbqAQC9Ck6UD+FWVlZTG4QqjvXgJxDlRRPtCkW0BsAFcEAqAohLrvOAgDFXSCYglFESkAAN13Ru3oi42SAAAAAElFTkSuQmCC>

[image7]: <data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABsAAAAXCAYAAAD6FjQuAAACeklEQVR4Xu2UT4iNURjGv6971Qghbjf33/nuvbghMl1kiiWRyN8o1qIZGxsrosleiIUpUUhZkgWLiSnKBoUiNWokKSuz0MT4Pfc7ZxzH3HJr7Oapp/Od57zve973fO85UTSNaXSCUqk0s1Kp7DXGXIIn6/V6ObQJkSRJV7VaXSc//BeH65MCB+Kbu3A3XGTH1xpDW6FYLJbY4Crr4x4/E6cR2obIYjgALzabzRlOJOstaB8JusI3RtvG2kv0TUwzjUZjDtpjOEx1Rd/2L5BNHsP38AUZL3C6SSv8QNATnrYHftLROc36v1GyTGOnt0NMRisV3BfdZlRxTXPGhPk7hiO+HfMD6F9ht6+3hW2OY6oODsE+5jXGEYLdxiTLeE4VqBLPNYN2kLXVntYe6iIcnhP8lDZFivneh3YTjqkyRnXQCDwb+v8z1IUEe0uQAaZZp7PZfLRnJu2y4/aofsKtnntHiO3RjBF8g7+Qy+Vmow/C77AHnoej2DV9u3agiFVKMHINozYlwDB8VSgUFvrGukcmPbbBeYDxTiebsdFRbA9PCHJUAHgjClo2Se/YD2wO8d3F973JTsBBJyGG+gTcZv49stAlv8Umj2q12lwJJr3047A/ChLDf7s9he58Pj8LvzN8X3G+LWiC+NT8eRn1H/fLuVwur3G2Jn2+1CCj6DucPd9rjfesMe7SNWC8D3ucfwsKaLO6YDvuOnxCIkt9Oz1jVHDZ/H4Hv8EvskVf7uzogyXEWY/+QB3tx2hBpdM9myvpi78MKRPaWMRKQnYE3AkTaaER8mkx1Kcc6moSHrLV9dpH4v9ADzmbPIT9/KKN4fqUQ/9Xv8bXfgHlu6eULLfGRgAAAABJRU5ErkJggg==>

[image8]: <data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAFsAAAAYCAYAAACV+oFbAAAFEUlEQVR4Xu2YWYhcRRSGu5kRIu7LOGaWru6Z0SGuSGswEFfiEtA8RB+UiPrghoiaCBGXh4AGCUREyWDcERlEnWBEjHkIOiYvah404JigBIkIgUAcCBoYJY7f31UN1afrtj3pEcnYP/zcvudU1b3116lz6nYu10YbbbTRxv8a5XL5uIGBgVOsvRG6urpOLBaL86y9jQaQyIj2SqlUWmh9jUB7B8Z0tb45B0VjoVC4FX7FpI/AP/h9c0a75/BvSkRiJ/YR7A8Ye1Po7++/grE3z3RXHFMoemxHqNckuASD03CtbYttEZyi/f3WR9/r8I23IFae/hsY53HrmBPo7e3tY4K7Ee8JbvOyKaKD2KtMcy3MGuz7aTMQ2/v6+o7HthXfQ7F9pgiLuRuWrO+YB5N6Fo7G0aioDWIvjdsi5mnYdsIxbjuNr4z9R7ggts8U0TNWWF+rYOybCIoLrf0/BRN9CR6iWF0c23nZxdj/TAkRFmjH8PDwSdbH7jlDCzc4ONgf27NOLbR9HY7mwk6bLfCON8KV1t4s0KNbu54xrtbpyfoFfPPkVzu1t/4adHd3n8BEtym6FGWxD9taeDAVvTzgbdHaKXqX0v5D+CTcywsMV33cPw+/tIIrZ2ctXCuQQFpIu+hNQLXkMXgY/uL8AeII812n9BnadPDeDwd9XsV3F9eP4YPqHw8m5BXJRZ+Tp+BP6hTxDbgfjttVDZMYd6aghjy+kesQoi9z/mUXyReli7oIDjVjH5wf22cDzHEhc3xRQWV9WQjvPlIVluvp3G9yPtW+o6DgnVfDL1QDo655nvUo9iWRrRKZl8Mf6PxbGOQAjX6uMkz+L5c4oVTFVkTGdkUQ9qeULri+66Io5vcCeLCYPtVI7LoibKEtS7v5R8F76Lud8c+zYybQqcVxpmBrHs5ngGn4GdxVSnwjKJU4r1lNjRMqRy/nt0rN4NyvCAPXFE0hS+wqNJbG1K6JbBrvMFFzWdS0giB2Xc2wKPhjarz7muUonKT/1pRAMYJYb9rdHHwuBOg0Yz1i/YL64Xu5p6fnzBqHDHSccInThvNFM5mv/0nssA2n8C+u2pwvghN1L5H7d9NIiMhRRLohl8ilFno/iZUSW2CclRIbTipFWX8Q+4WCqX+a5BLnE/9tsT0qmipwqQqrL8cxCWgdghYhFk8PdhlHSMH5qM96VivQzn0a3mIdDaC5rXeJc3/I/4rsD+C0ftudwlzLcGPOzDMzhYSttNeF4jg0NHQyA6wu1Z4sFPmfKI/GfYNvlYvE1mSdX9S6fC04f3JJjtUK2GEXMO5ILrHAjRBEXafaU7Up/THW93B5qElakGn4De0vokmeQnpOSFW1ER8E1ZfbBjWMfRIpiLWPh97tfEHYEm+tkCq+S6UFrTbYBT+l/3tcJ+HvWnXbNhd2SVZKagWMea/e09qbAe+0FH7rfL7fAyeYy1VRkw7ub8d+wHnRRf2uq3Ea7Bqo8+HZ1hf+/vwI/yE9kJe+MzpfVuB8EdwT5+WAfDgvd3I6OUvHJtq85RLnayGMo0WvHBFnE0qHcXQeBTo0B7Q41Toi5PURp3b87rDO2YLS0DPO7AznPwZ0ZKysMNdL4K/FjH8GWYg7FP0tijL3Ef7U+hzBzq/anD917JSv6PE19+tTYobP+i11Oa6NNCQUgr1fTRFcz1WRwLYDbsN/Zc7UhAB9ba3RTtBv62wjAwh2bSHjkJ8FFuF6+tyXm8NC/w0tY4KUpLoiogAAAABJRU5ErkJggg==>

[image9]: <data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAABcAAAAXCAYAAADgKtSgAAABL0lEQVR4XmNgGAWjgGggJydXLi8v/59EfEBUVJQH3SycAKihCqQRaJkvuhxQzBgo9xWID6urq/OiyxMEMB8QMJw0F8PAqOFYATUMB8kpKCgIoItTYjgjUNwJqOYgkH4FUqeoqGiGooJMwxmBLi0AyseB2EBaCajmBBArIqkhz3BZWVk/oNwMY2NjVhAfKJ8DtGwCkMkI1wwCSIaXo0hA5DAMV1JS4gfy9wHlbGDqoBmMGa4RBuQROZQow0GGAl15SkpKSgRdPRwAFWTIQyLiPxJ+BtTsIiMjowtkXwHiv0hyX4ByK4D68pAtoyqABiN6WcMI5aOGOakAaHA0EH8F+sAcKgRKkqVAXISikBwAjdA9QPwRaMFCoE82AHEoA6WuhgFQEgQmRxOgodqw5IgMAP9VicigckuyAAAAAElFTkSuQmCC>

[image10]: <data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAmwAAAAxCAYAAABnGvUlAAAJS0lEQVR4Xu3cf2heVx3H8Sek6vzt1FqXtM9J0mhoRaZU56+p+2O6leLQWUHcUFBxOAbDTRjKnMoo6KQyhn+MulEVRrd1IIJDHWNE9I+gZXNg6dCJnRSGyhBlFVS6+Pk893uSk5P7PMuSNG2a9wsO99zz495z7/OU8+25N0+nAwAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAMEBK6Q6nuhwAAGwwIyMjr9dmuC7HmaVA7e3aDHe73fcpXVzXAwCADUSBwczExMSr6/JzlYKfXbG9Sekjdf2ZkMcU+d6YIpAeGhsbO2+tVtm2b9/+hl27dr2oLl+vNm/e/IpOn/+M6D4fHPS91z2/Uff+S3U5AABrTpPW+Q5c6vL1Ynx8/MOaWH9alw+yXgK2si5W204rBSc/0XlOlmMpqfwHddnZTNfyH6XZLVu2vLyuGx0d3arreUtdXnLgqv4/qstXg859vY59QOmCug4AgEU0YVylzaa6fL2YmJh4s67ha3X5IOspYNO1fUdpx9atW9+Uy06nQQGbV5zqsqXwfa7LMgWJ1wyqXwkd91KN+VRdbip/oi7rY5PG+I26MJucnHyVjjWjNh+t6/pR+7s68W9O+X0a58+rJgCADczvQu0qH3ft3LnzxZpo3lY2WgXDmoSujUm/9yjK5xwfH79I53qNV8RUNORyP45ycDI6Ovo6B13l2HIfb9VvLB/PbbV7iR8Tup1XSuKYffvkY2aDAjZ1u8yTcNuqzHLp2nZ73Hk/HtMt0Bawqd/nlGYjXTnf+oXRPfmAx6DthXGPtvgeqmrYj0BVt0cB4WvdNkXA5jGU98CfXXWvhnw8lX+mKPPn4c9yn++h97vNStKtqVlJqj8Lf1dOud7jyIXKb/NxHYyXjV8oHfdBpWlfa3n//d1R+VfLtjIU5+uNUfdjNFeo7VPbtm1751zLPnTcT6rto8/33VGb53Le91T7s2U9AGAD02Ryuycppf3e12Q75QmmbrdCft/q9gjO/HjtZy5Ufiwm5vuV7tEk9SFt9yidUP4L2j4ZE/vc46HoM6vyO7W9QelPSncpfU9VD2n7uNtp+4TbDehzIB8z6xewqftl7u9+yn9lvsfy6VhX6ljfzuNV/jyl++p2bQHbanAgpGDjCp3zch33L75H2v4yNYGMg7B7U7GqpvxzSo9pf6+2zyjtifJrfW/ycVV/UPs/1vbTvsYoc/6vSuNKxxxMu43SEaUDPl/ub6kJSH2+I6r7lsviMeQjDga1PfR8wc8g6n9c6TGlqzyubjwCVf6C+h7HZzSrdI/SDo8912n/qbT0gNlB6EwOWNv4PDnvcZT7AIANLibA94xFkKb8HfFS+6pyQOKtJ0VPdEX5D12W91MTJPQehXmr/d25LnOfPMZo8+/I+52uk863nafq02tXysGJ6+uJW4ZU9om2F+99bXG+RalcwSmp7vNKM0qHvK9jX+qk7FAZjOQxRb4e07L5XD63g6X82ZjKpot8GbA54O2d36tKKe551PXus++v8g8Wn/VJtX1HagK8HS7Lq4g+ru9zPkYtzt2rj+Mercb5tOon5nssnfr+LT9KTs37bL3vWIyp32Pf2+LzKcumlfaVZf3EI9Ijbd+fLBGwAQD68QTqiacz/+7Mvxa2WGQotQQmOTkAqDvEOe6PiX5RIFU+Cuw2K2r/0PYWbU/k8lLZx5N6iuDLk23ODzpP2aeUJ2vX5+AkDKv/F7XtPbJdLSmCDgdoyj8cgZ9XAOf+8rMMIMoxuU3Zbjn8CFTHOJyawKB3bWkJAVtxn3Of3n3WdofSA53i3cf47L0SteAleh/D99n58jFjFuf25+RjOj3TKY6r/f91l/GTJj5XiiArxjY33hjTgoDNj4TV5vtlWaby6XwNbXzNqVn5fVdnCd8dX1PO+14nAjYAQBaTylxgU04abbxCoMlkb78Uk9MC3WY1p/d+Tj6fy5zqgM378cjMwd/cikqpX/DlyTbn6+vq16eUJ2vXe8LM5al5qb434eo438zlK6XjTntMOteE8k/r2GPa/tnlfo/QbfKYIp/fYVvx77AV1+cA/NZ8b3zu3Mb3KJ8/LQzYLk7F9yTfZwU3L1X+t6o/3/tevZ2cnNzstnmcKksOmnxc3+c43qKVwzi3P4eb/E6j9mempqZeGdWbtH/MAeeCTkvgcfhRsPMpVpb9eHhkZORl3eZzWLCiq7I7Nd5J592u/K76WpVumG89T2ObUt1h9d9Z1/Wj9sdzXuO6JjVBKgAAPZ6wb1S6Oybbg3WDlYqX/f0ukCewWzwJK/8rTZIO5GaVnlXZvW7bbVbY/P6Sy51+Xx5Lk+dbiz5XF+2+q/TfyHvV6NmcV5+L+vUpj91tCdg0ub8/ApG/u4+u441ln5XQ8U4oPZCacR93mc77iyIwaQ3YVuN32Hws9X1U6bCO81Au1/4fU/OzEofiHvVWeWKcn426aQeXRZ8yMP5gHOPusfjDFb+0r/xvouzyaOrv3T993LbHhKn5Trp+fxS5/e9S89k+6f2y/RJ5pfQ+f57eUVDl74Wv67ao9znm7qfyNyvgfIm3qbkXN+e6qD/atjq4XA5MPT4dd7fSr8v35QAAG5wnfSevVqTiXbbTIS18LNY64arN43lFo9NMoEcXNDiN2gK2/D6Z71G/99GWaTj/ZWZq3oXqvctWBgzWFrCVdWmZv8MWK3gOYOpH2ENxnUMeWx1Muazct1QEbGGo/OvOrO7r1apB9zRWs8rvSR5b/VelS1aukFm+1rzvwDLny8C57mfxma3qT9/4M1Xa21nBNQIAzkGp+eu96zRJXK/J6g91/VpLzSPBj8WuV5HWbExtAdvpomt82Csq8V7fdZ0maHCAuj8VKzn9Ara0xr/D1kbnP6XP5xJtj9V165Ufl+YVuEH0WVxdB7Ol1KxELkrq9/W24A8AACxR/n0vTawfHx8fv7CuPxPK3xzLY0qr9DtsaKd7+mUFou+uyzPV7+NxJQAAAAAAOLd1mx/dnfZvgNV1AAAAODv43bdFPy8CAACAs0D8Tpz/evK9Sp+q6wEAAHCG+ec3OvEXl3UdAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAcO74Px+HZItDBMn3AAAAAElFTkSuQmCC>

[image11]: <data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAA8AAAAYCAYAAAAlBadpAAABeUlEQVR4Xu2SP0vEQBDFc1wEQVEUz+Bd/udAO4VgIRx2go294AcQQRC0EMTC1kIQGyuxshDuI4iIjYWtciBYCDbXWthY6G9wI5uNXK6w9MGwmXlvMjO7Y1n/KEO1Xq9PpGk6YBKl8H2/FQTBB+eKyZWCxG3sFYtMrgw2SW2qnvNdMcmeiKLIIbkThuGyyZVCzXsdx/GoyVUhU/6+aN6kiCVGxXWSN3XOUsQxxB7nLedRxvGzafwu8VVsxnXdcT1X2llCsN9sNkc4bxBdEq4Kh7+GvXueN59LyoB4gwqziBZEKFUyDv8Ee5TF0HMKIOkg0N6Qjsb4vscurF5PU6vVhqVlrI1rS0x18yYXZcjzQDSFvVBtV4vJvLKKLV1bgFqA5yxZngf/qq95QQXhjpr5TM362dcqMtegmMyuupCbl3l/bv5XNBoNF2FH2nQcZ4iQTcVT/IckSSZNfQ7qVrvB9+pJ4hYVn7A5U1uArKea906MpEPZNlP35/gC869WFFlxbl8AAAAASUVORK5CYII=>

[image12]: <data:image/png;base64,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>