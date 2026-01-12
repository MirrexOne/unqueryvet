package dev.unqueryvet

import com.intellij.lang.annotation.AnnotationHolder
import com.intellij.lang.annotation.ExternalAnnotator
import com.intellij.lang.annotation.HighlightSeverity
import com.intellij.openapi.editor.Editor
import com.intellij.openapi.util.TextRange
import com.intellij.psi.PsiFile
import java.io.BufferedReader
import java.io.InputStreamReader
import java.util.regex.Pattern

/**
 * External annotator that runs unqueryvet on Go files.
 */
class UnqueryvetAnnotator : ExternalAnnotator<PsiFile, List<UnqueryvetIssue>>() {

    override fun collectInformation(file: PsiFile): PsiFile? {
        if (file.virtualFile?.extension != "go") {
            return null
        }
        return file
    }

    override fun collectInformation(file: PsiFile, editor: Editor, hasErrors: Boolean): PsiFile? {
        return collectInformation(file)
    }

    override fun doAnnotate(collectedInfo: PsiFile?): List<UnqueryvetIssue>? {
        val file = collectedInfo ?: return null
        val path = file.virtualFile?.path ?: return null

        val settings = UnqueryvetSettings.getInstance()
        if (!settings.enabled) {
            return emptyList()
        }

        return runUnqueryvet(path, settings)
    }

    override fun apply(file: PsiFile, issues: List<UnqueryvetIssue>?, holder: AnnotationHolder) {
        issues ?: return

        for (issue in issues) {
            val startOffset = lineColumnToOffset(file.text, issue.line, issue.column)
            val endOffset = if (issue.endColumn > 0) {
                lineColumnToOffset(file.text, issue.line, issue.endColumn)
            } else {
                findEndOfString(file.text, startOffset)
            }

            val severity = when (issue.severity) {
                "error" -> HighlightSeverity.ERROR
                "warning" -> HighlightSeverity.WARNING
                else -> HighlightSeverity.WEAK_WARNING
            }

            holder.newAnnotation(severity, issue.message)
                .range(TextRange(startOffset, endOffset))
                .withFix(ReplaceSelectStarQuickFix(issue))
                .create()
        }
    }

    private fun runUnqueryvet(path: String, settings: UnqueryvetSettings): List<UnqueryvetIssue> {
        val issues = mutableListOf<UnqueryvetIssue>()

        try {
            val command = mutableListOf(settings.binaryPath, "-json")

            if (settings.enableN1Detection) {
                command.add("-n1")
            }
            if (settings.enableSQLiDetection) {
                command.add("-sqli")
            }

            command.add(path)

            val process = ProcessBuilder(command)
                .redirectErrorStream(true)
                .start()

            val reader = BufferedReader(InputStreamReader(process.inputStream))
            val output = reader.readText()
            reader.close()

            process.waitFor()

            // Parse JSON output
            issues.addAll(parseOutput(output))

        } catch (e: Exception) {
            // Log error but don't crash
        }

        return issues
    }

    private fun parseOutput(output: String): List<UnqueryvetIssue> {
        val issues = mutableListOf<UnqueryvetIssue>()
        val pattern = Pattern.compile("(\\S+):(\\d+):(\\d+):\\s+(.*)")

        for (line in output.lines()) {
            val matcher = pattern.matcher(line)
            if (matcher.matches()) {
                issues.add(UnqueryvetIssue(
                    file = matcher.group(1),
                    line = matcher.group(2).toInt(),
                    column = matcher.group(3).toInt(),
                    endColumn = 0,
                    message = matcher.group(4),
                    severity = if (line.contains("SELECT *")) "warning" else "info"
                ))
            }
        }

        return issues
    }

    private fun lineColumnToOffset(text: String, line: Int, column: Int): Int {
        var currentLine = 1
        var offset = 0

        for (char in text) {
            if (currentLine == line) {
                return offset + column - 1
            }
            if (char == '\n') {
                currentLine++
            }
            offset++
        }

        return offset
    }

    private fun findEndOfString(text: String, startOffset: Int): Int {
        if (startOffset >= text.length) return startOffset

        val quote = text[startOffset]
        if (quote != '"' && quote != '`') {
            // Find end of word
            var end = startOffset
            while (end < text.length && !text[end].isWhitespace()) {
                end++
            }
            return end
        }

        var i = startOffset + 1
        while (i < text.length) {
            if (text[i] == quote && (i == 0 || text[i - 1] != '\\')) {
                return i + 1
            }
            i++
        }

        return text.length
    }
}

/**
 * Represents an issue found by unqueryvet.
 */
data class UnqueryvetIssue(
    val file: String,
    val line: Int,
    val column: Int,
    val endColumn: Int,
    val message: String,
    val severity: String
)
