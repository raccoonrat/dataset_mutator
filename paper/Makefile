# 在 paper/ 目录生成 PDF（输出文件位于本目录）
# 用法：make / make pdf（中文版）| make pdf-en（英文版）
# 依赖：pdflatex、bibtex；中文版另需 CJK

MAIN_CN := BeyondModelReflection_DecoupledSafety_CN
MAIN_EN := BeyondModelReflection_DecoupledSafety
LATEX := pdflatex
BIB := bibtex

.PHONY: pdf pdf-en clean cleanall

# 默认与历史兼容：make == 中文版
MAIN ?= $(MAIN_CN)
pdf: $(MAIN_CN).pdf

pdf-en: $(MAIN_EN).pdf

$(MAIN_CN).pdf: $(MAIN_CN).tex usenix-2020-09.sty ArchitecturalReconstruction_ControlledMutation.bib
	$(LATEX) -interaction=nonstopmode -file-line-error $(MAIN_CN).tex
	$(BIB) $(MAIN_CN)
	$(LATEX) -interaction=nonstopmode -file-line-error $(MAIN_CN).tex
	$(LATEX) -interaction=nonstopmode -file-line-error $(MAIN_CN).tex

$(MAIN_EN).pdf: $(MAIN_EN).tex usenix-2020-09.sty ArchitecturalReconstruction_ControlledMutation.bib
	$(LATEX) -interaction=nonstopmode -file-line-error $(MAIN_EN).tex
	$(BIB) $(MAIN_EN)
	$(LATEX) -interaction=nonstopmode -file-line-error $(MAIN_EN).tex
	$(LATEX) -interaction=nonstopmode -file-line-error $(MAIN_EN).tex

clean:
	rm -f $(MAIN_CN).aux $(MAIN_CN).bbl $(MAIN_CN).blg $(MAIN_CN).log $(MAIN_CN).out $(MAIN_CN).toc $(MAIN_CN).pdf
	rm -f $(MAIN_EN).aux $(MAIN_EN).bbl $(MAIN_EN).blg $(MAIN_EN).log $(MAIN_EN).out $(MAIN_EN).toc $(MAIN_EN).pdf

cleanall: clean
	@true
