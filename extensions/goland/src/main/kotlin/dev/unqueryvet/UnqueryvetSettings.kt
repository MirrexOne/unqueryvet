package dev.unqueryvet

import com.intellij.openapi.application.ApplicationManager
import com.intellij.openapi.components.PersistentStateComponent
import com.intellij.openapi.components.State
import com.intellij.openapi.components.Storage

/**
 * Persistent settings for unqueryvet plugin.
 */
@State(
    name = "UnqueryvetSettings",
    storages = [Storage("unqueryvet.xml")]
)
class UnqueryvetSettings : PersistentStateComponent<UnqueryvetSettings.State> {

    data class State(
        var enabled: Boolean = true,
        var binaryPath: String = "unqueryvet",
        var enableN1Detection: Boolean = true,
        var enableSQLiDetection: Boolean = true,
        var enableComplexityAnalysis: Boolean = false,
        var complexityThreshold: Int = 25,
        var autoFix: Boolean = false,
        var excludePatterns: MutableList<String> = mutableListOf(
            "*_test.go",
            "vendor/**",
            "testdata/**"
        )
    )

    private var myState = State()

    override fun getState(): State = myState

    override fun loadState(state: State) {
        myState = state
    }

    var enabled: Boolean
        get() = myState.enabled
        set(value) { myState.enabled = value }

    var binaryPath: String
        get() = myState.binaryPath
        set(value) { myState.binaryPath = value }

    var enableN1Detection: Boolean
        get() = myState.enableN1Detection
        set(value) { myState.enableN1Detection = value }

    var enableSQLiDetection: Boolean
        get() = myState.enableSQLiDetection
        set(value) { myState.enableSQLiDetection = value }

    var enableComplexityAnalysis: Boolean
        get() = myState.enableComplexityAnalysis
        set(value) { myState.enableComplexityAnalysis = value }

    var complexityThreshold: Int
        get() = myState.complexityThreshold
        set(value) { myState.complexityThreshold = value }

    var autoFix: Boolean
        get() = myState.autoFix
        set(value) { myState.autoFix = value }

    var excludePatterns: MutableList<String>
        get() = myState.excludePatterns
        set(value) { myState.excludePatterns = value }

    companion object {
        fun getInstance(): UnqueryvetSettings {
            return ApplicationManager.getApplication().getService(UnqueryvetSettings::class.java)
        }
    }
}
