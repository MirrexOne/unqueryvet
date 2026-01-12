package dev.unqueryvet

import com.intellij.codeInsight.intention.IntentionAction
import com.intellij.codeInsight.intention.PsiElementBaseIntentionAction
import com.intellij.codeInspection.LocalQuickFix
import com.intellij.codeInspection.ProblemDescriptor
import com.intellij.openapi.editor.Editor
import com.intellij.openapi.project.Project
import com.intellij.psi.PsiElement
import com.intellij.psi.PsiFile
import com.intellij.openapi.util.Iconable

/**
 * Quick fix to replace SELECT * with specific columns.
 * Implements IntentionAction for use with AnnotationBuilder.withFix()
 */
class ReplaceSelectStarQuickFix(private val issue: UnqueryvetIssue) : IntentionAction {

    override fun getText(): String = "Replace SELECT * with specific columns"

    override fun getFamilyName(): String = "unqueryvet"

    override fun isAvailable(project: Project, editor: Editor?, file: PsiFile?): Boolean = true

    override fun invoke(project: Project, editor: Editor?, file: PsiFile?) {
        val document = file?.viewProvider?.document ?: return

        // Try to determine columns from context
        val suggestedColumns = extractSuggestedColumns(issue.message)

        // Find the SELECT * in the line
        val lineStartOffset = document.getLineStartOffset(issue.line - 1)
        val lineEndOffset = document.getLineEndOffset(issue.line - 1)
        val lineText = document.getText(com.intellij.openapi.util.TextRange(lineStartOffset, lineEndOffset))

        val replacement = if (suggestedColumns.isNotEmpty()) {
            "SELECT " + suggestedColumns.joinToString(", ")
        } else {
            "SELECT /* TODO: specify columns */"
        }

        val newLineText = lineText.replace(
            Regex("SELECT\\s+\\*", RegexOption.IGNORE_CASE),
            replacement
        )

        document.replaceString(lineStartOffset, lineEndOffset, newLineText)
    }

    override fun startInWriteAction(): Boolean = true

    private fun extractSuggestedColumns(message: String): List<String> {
        // Extract columns from message like "suggest: id, name, email"
        val suggestPattern = Regex("suggest:\\s*([\\w,\\s]+)")
        val match = suggestPattern.find(message)

        return match?.groupValues?.getOrNull(1)
            ?.split(",")
            ?.map { it.trim() }
            ?.filter { it.isNotEmpty() }
            ?: emptyList()
    }
}

/**
 * LocalQuickFix version for use with inspections.
 */
class ReplaceSelectStarLocalQuickFix(private val issue: UnqueryvetIssue) : LocalQuickFix {

    override fun getName(): String = "Replace SELECT * with specific columns"

    override fun getFamilyName(): String = "unqueryvet"

    override fun applyFix(project: Project, descriptor: ProblemDescriptor) {
        val element = descriptor.psiElement ?: return

        val suggestedColumns = extractSuggestedColumns(issue.message)

        if (suggestedColumns.isNotEmpty()) {
            replaceSelectStar(element, suggestedColumns)
        } else {
            showColumnSelectionDialog(project, element)
        }
    }

    private fun extractSuggestedColumns(message: String): List<String> {
        val suggestPattern = Regex("suggest:\\s*([\\w,\\s]+)")
        val match = suggestPattern.find(message)

        return match?.groupValues?.getOrNull(1)
            ?.split(",")
            ?.map { it.trim() }
            ?.filter { it.isNotEmpty() }
            ?: emptyList()
    }

    private fun replaceSelectStar(element: PsiElement, columns: List<String>) {
        val text = element.text
        val newText = text.replace(
            Regex("SELECT\\s+\\*", RegexOption.IGNORE_CASE),
            "SELECT " + columns.joinToString(", ")
        )

        val document = element.containingFile?.viewProvider?.document ?: return
        val startOffset = element.textRange.startOffset
        val endOffset = element.textRange.endOffset

        document.replaceString(startOffset, endOffset, newText)
    }

    private fun showColumnSelectionDialog(project: Project, element: PsiElement) {
        val text = element.text
        val newText = text.replace(
            Regex("SELECT\\s+\\*", RegexOption.IGNORE_CASE),
            "SELECT /* TODO: specify columns */"
        )

        val document = element.containingFile?.viewProvider?.document ?: return
        val startOffset = element.textRange.startOffset
        val endOffset = element.textRange.endOffset

        document.replaceString(startOffset, endOffset, newText)
    }
}

/**
 * Intention action for replacing SELECT *.
 */
class ReplaceSelectStarIntention : PsiElementBaseIntentionAction(), IntentionAction {

    override fun getFamilyName(): String = "unqueryvet"

    override fun getText(): String = "Replace SELECT * with specific columns"

    override fun isAvailable(project: Project, editor: Editor?, element: PsiElement): Boolean {
        // Check if cursor is on a SQL string containing SELECT *
        val text = element.text
        return text.contains("SELECT *", ignoreCase = true) ||
               text.contains("SELECT\t*", ignoreCase = true)
    }

    override fun invoke(project: Project, editor: Editor?, element: PsiElement) {
        val text = element.text

        // Simple replacement - in production, would analyze struct to suggest columns
        val newText = text.replace(
            Regex("SELECT\\s+\\*", RegexOption.IGNORE_CASE),
            "SELECT /* specify columns */"
        )

        val document = element.containingFile?.viewProvider?.document ?: return
        val startOffset = element.textRange.startOffset
        val endOffset = element.textRange.endOffset

        document.replaceString(startOffset, endOffset, newText)
    }
}
