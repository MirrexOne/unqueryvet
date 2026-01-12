package dev.unqueryvet

import com.intellij.codeInspection.*
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiElementVisitor
import com.intellij.psi.PsiFile

/**
 * Inspection that detects SELECT * usage in Go SQL strings.
 */
class SelectStarInspection : LocalInspectionTool() {

    override fun getDisplayName(): String = "SELECT * usage in SQL"

    override fun getGroupDisplayName(): String = "SQL"

    override fun getShortName(): String = "SelectStarUsage"

    override fun isEnabledByDefault(): Boolean = true

    override fun buildVisitor(holder: ProblemsHolder, isOnTheFly: Boolean): PsiElementVisitor {
        return object : PsiElementVisitor() {
            override fun visitElement(element: PsiElement) {
                if (isStringLiteral(element)) {
                    checkForSelectStar(element, holder)
                }
            }
        }
    }

    private fun isStringLiteral(element: PsiElement): Boolean {
        val elementType = element.node?.elementType?.toString() ?: return false
        return elementType == "STRING" ||
               elementType == "RAW_STRING" ||
               elementType.contains("STRING_LITERAL")
    }

    private fun checkForSelectStar(element: PsiElement, holder: ProblemsHolder) {
        val text = element.text

        // Check for SELECT * pattern
        val selectStarPattern = Regex("""SELECT\s+\*""", RegexOption.IGNORE_CASE)

        if (selectStarPattern.containsMatchIn(text)) {
            holder.registerProblem(
                element,
                "SELECT * usage detected - specify columns explicitly",
                ProblemHighlightType.WARNING,
                createQuickFix()
            )
        }

        // Also check for table.* pattern
        val tableStarPattern = Regex("""SELECT\s+\w+\.\*""", RegexOption.IGNORE_CASE)

        if (tableStarPattern.containsMatchIn(text)) {
            holder.registerProblem(
                element,
                "SELECT table.* usage detected - specify columns explicitly",
                ProblemHighlightType.WARNING,
                createQuickFix()
            )
        }
    }

    private fun createQuickFix(): LocalQuickFix {
        return object : LocalQuickFix {
            override fun getName(): String = "Replace with specific columns"

            override fun getFamilyName(): String = "unqueryvet"

            override fun applyFix(project: com.intellij.openapi.project.Project, descriptor: ProblemDescriptor) {
                val element = descriptor.psiElement ?: return
                val text = element.text

                val newText = text.replace(
                    Regex("""SELECT\s+\*""", RegexOption.IGNORE_CASE),
                    "SELECT /* TODO: specify columns */"
                )

                val document = element.containingFile?.viewProvider?.document ?: return
                document.replaceString(
                    element.textRange.startOffset,
                    element.textRange.endOffset,
                    newText
                )
            }
        }
    }

    override fun checkFile(file: PsiFile, manager: InspectionManager, isOnTheFly: Boolean): Array<ProblemDescriptor>? {
        if (file.virtualFile?.extension != "go") {
            return null
        }
        return super.checkFile(file, manager, isOnTheFly)
    }
}
